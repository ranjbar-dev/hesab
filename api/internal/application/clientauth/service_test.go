// ponytail: parallel to adminauth, kept separate on purpose.
package clientauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/config"
	"hesab/api/internal/domain/user"
	"hesab/api/internal/infrastructure/token"
)

var errNotFound = errors.New("not found")

type fakeRepo struct {
	users   map[int64]*user.User
	refresh map[string]*RefreshToken
	resets  []*resetRow
	nextID  int64
}
type resetRow struct {
	id, userID int64
	codeHash   string
	expiresAt  time.Time
	consumedAt *time.Time
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[int64]*user.User{}, refresh: map[string]*RefreshToken{}, nextID: 1}
}
func (f *fakeRepo) UserByPhone(_ context.Context, p string) (user.User, error) {
	for _, u := range f.users {
		if u.PhoneNumber == p {
			return *u, nil
		}
	}
	return user.User{}, errNotFound
}
func (f *fakeRepo) UserByID(_ context.Context, id int64) (user.User, error) {
	if u, ok := f.users[id]; ok {
		return *u, nil
	}
	return user.User{}, errNotFound
}
func (f *fakeRepo) UpdatePassword(_ context.Context, id int64, h string) error {
	f.users[id].PasswordHash = h
	return nil
}
func (f *fakeRepo) SetTOTPSecret(_ context.Context, id int64, s string) error {
	f.users[id].TOTPSecret = s
	return nil
}
func (f *fakeRepo) InsertRefreshToken(_ context.Context, id int64, h string, e time.Time) error {
	f.refresh[h] = &RefreshToken{ID: f.nextID, UserID: id, Hash: h, ExpiresAt: e}
	f.nextID++
	return nil
}
func (f *fakeRepo) RefreshTokenByHash(_ context.Context, h string) (RefreshToken, error) {
	if r, ok := f.refresh[h]; ok {
		return *r, nil
	}
	return RefreshToken{}, errNotFound
}
func (f *fakeRepo) RevokeRefreshToken(_ context.Context, h string) error {
	if r, ok := f.refresh[h]; ok && r.RevokedAt == nil {
		n := time.Now()
		r.RevokedAt = &n
	}
	return nil
}
func (f *fakeRepo) RevokeAllRefreshTokens(_ context.Context, id int64) error {
	for h := range f.refresh {
		_ = f.RevokeRefreshToken(context.Background(), h)
	}
	return nil
}
func (f *fakeRepo) InvalidatePasswordResets(_ context.Context, id int64) error {
	for _, r := range f.resets {
		if r.userID == id && r.consumedAt == nil {
			n := time.Now()
			r.consumedAt = &n
		}
	}
	return nil
}
func (f *fakeRepo) InsertPasswordReset(_ context.Context, id int64, h string, e time.Time) error {
	f.resets = append(f.resets, &resetRow{id: f.nextID, userID: id, codeHash: h, expiresAt: e})
	f.nextID++
	return nil
}
func (f *fakeRepo) LatestPasswordReset(_ context.Context, id int64) (PasswordReset, error) {
	var found *resetRow
	for _, r := range f.resets {
		if r.userID == id && r.consumedAt == nil && r.expiresAt.After(time.Now()) {
			found = r
		}
	}
	if found == nil {
		return PasswordReset{}, errNotFound
	}
	return PasswordReset{ID: found.id, UserID: found.userID, CodeHash: found.codeHash, ExpiresAt: found.expiresAt}, nil
}
func (f *fakeRepo) ConsumePasswordReset(_ context.Context, id int64) error {
	for _, r := range f.resets {
		if r.id == id {
			n := time.Now()
			r.consumedAt = &n
		}
	}
	return nil
}

type fakeSMS struct{ lastPhone, lastMsg string }

func (s *fakeSMS) Send(_ context.Context, p, m string) error {
	s.lastPhone, s.lastMsg = p, m
	return nil
}
func newSvc(t *testing.T) (*Service, *fakeRepo, *user.User) {
	t.Helper()
	r := newFakeRepo()
	h, e := bcrypt.GenerateFromPassword([]byte("Client@Pass1999"), bcrypt.MinCost)
	if e != nil {
		t.Fatal(e)
	}
	u := &user.User{ID: 1, FirstName: "تست", LastName: "کاربر", Email: "user@hesab.local", PhoneNumber: "9120000000", PasswordHash: string(h)}
	r.users[1] = u
	cfg := config.Config{JWTSecret: "test-secret", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 720 * time.Hour, PasswordResetTTL: 5 * time.Minute, TwoFAPendingTTL: 5 * time.Minute, ClientTOTPIssuer: "Hesab"}
	return NewService(r, clientTokens{token.New(cfg)}, &fakeSMS{}, func() string { return "123456" }, time.Now, cfg), r, u
}

type clientTokens struct{ token.JWT }

func (t clientTokens) IssueAccess(id int64) (string, int, error) { return t.IssueClientAccess(id) }
func (t clientTokens) IssuePending(id int64) (string, error)     { return t.IssueClientPending(id) }
func (t clientTokens) ParseAccess(s string) (int64, error)       { return t.ParseClientAccess(s) }
func (t clientTokens) ParsePending(s string) (int64, error)      { return t.ParseClientPending(s) }
func TestClientLoginRefreshLogout(t *testing.T) {
	s, _, _ := newSvc(t)
	r, e := s.Login(context.Background(), "09120000000", "Client@Pass1999")
	if e != nil || r.AccessToken == "" || r.RefreshToken == "" {
		t.Fatalf("login: %+v %v", r, e)
	}
	next, e := s.Refresh(context.Background(), r.RefreshToken)
	if e != nil || next.RefreshToken == r.RefreshToken {
		t.Fatalf("refresh: %+v %v", next, e)
	}
	if _, e = s.Refresh(context.Background(), r.RefreshToken); !errors.Is(e, user.ErrRefreshInvalid) {
		t.Fatalf("old token accepted: %v", e)
	}
	_ = s.Logout(context.Background(), next.RefreshToken)
	if _, e = s.Refresh(context.Background(), next.RefreshToken); !errors.Is(e, user.ErrRefreshInvalid) {
		t.Fatalf("logout token accepted: %v", e)
	}
}
func TestClientPasswordResetAndTwoFA(t *testing.T) {
	s, _, u := newSvc(t)
	if e := s.ForgotPassword(context.Background(), u.PhoneNumber); e != nil {
		t.Fatal(e)
	}
	if e := s.ResetPassword(context.Background(), u.PhoneNumber, "123456", "NewPass2026"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Login(context.Background(), u.PhoneNumber, "NewPass2026"); e != nil {
		t.Fatal(e)
	}
	secret, _, e := s.Setup2FA(context.Background(), u.ID)
	if e != nil {
		t.Fatal(e)
	}
	code, _ := totp.GenerateCode(secret, time.Now())
	if e = s.Activate2FA(context.Background(), u.ID, secret, code); e != nil {
		t.Fatal(e)
	}
	r, e := s.Login(context.Background(), u.PhoneNumber, "NewPass2026")
	if e != nil || !r.TwoFARequired {
		t.Fatalf("want 2fa: %+v %v", r, e)
	}
	code, _ = totp.GenerateCode(secret, time.Now())
	if _, e = s.LoginVerify2FA(context.Background(), r.PendingToken, code); e != nil {
		t.Fatal(e)
	}
}

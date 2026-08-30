package adminauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"hesab/api/internal/config"
	"hesab/api/internal/domain/admin"
	"hesab/api/internal/infrastructure/token"
)

var errNotFound = errors.New("not found")

// fakeRepo is an in-memory Repository for service-level tests.
type fakeRepo struct {
	admins  map[int64]*admin.Admin
	refresh map[string]*RefreshToken
	resets  []*resetRow
	nextID  int64
}

type resetRow struct {
	id, adminID int64
	codeHash    string
	expiresAt   time.Time
	consumedAt  *time.Time
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{admins: map[int64]*admin.Admin{}, refresh: map[string]*RefreshToken{}, nextID: 1}
}

func (f *fakeRepo) AdminByPhone(_ context.Context, p string) (admin.Admin, error) {
	for _, a := range f.admins {
		if a.PhoneNumber == p {
			return *a, nil
		}
	}
	return admin.Admin{}, errNotFound
}

func (f *fakeRepo) AdminByID(_ context.Context, id int64) (admin.Admin, error) {
	if a, ok := f.admins[id]; ok {
		return *a, nil
	}
	return admin.Admin{}, errNotFound
}

func (f *fakeRepo) UpdatePassword(_ context.Context, id int64, hash string) error {
	f.admins[id].PasswordHash = hash
	return nil
}

func (f *fakeRepo) SetTOTPSecret(_ context.Context, id int64, s string) error {
	f.admins[id].TOTPSecret = s
	return nil
}

func (f *fakeRepo) InsertRefreshToken(_ context.Context, adminID int64, h string, exp time.Time) error {
	f.refresh[h] = &RefreshToken{ID: f.nextID, AdminID: adminID, Hash: h, ExpiresAt: exp}
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
		now := time.Now()
		r.RevokedAt = &now
	}
	return nil
}

func (f *fakeRepo) RevokeAllRefreshTokens(_ context.Context, adminID int64) error {
	now := time.Now()
	for _, r := range f.refresh {
		if r.AdminID == adminID && r.RevokedAt == nil {
			r.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) InvalidatePasswordResets(_ context.Context, adminID int64) error {
	now := time.Now()
	for _, r := range f.resets {
		if r.adminID == adminID && r.consumedAt == nil {
			r.consumedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) InsertPasswordReset(_ context.Context, adminID int64, codeHash string, exp time.Time) error {
	f.resets = append(f.resets, &resetRow{id: f.nextID, adminID: adminID, codeHash: codeHash, expiresAt: exp})
	f.nextID++
	return nil
}

func (f *fakeRepo) LatestPasswordReset(_ context.Context, adminID int64) (PasswordReset, error) {
	var found *resetRow // slice is in insertion order, so the last match is newest
	for _, r := range f.resets {
		if r.adminID == adminID && r.consumedAt == nil && r.expiresAt.After(time.Now()) {
			found = r
		}
	}
	if found == nil {
		return PasswordReset{}, errNotFound
	}
	return PasswordReset{ID: found.id, AdminID: found.adminID, CodeHash: found.codeHash, ExpiresAt: found.expiresAt}, nil
}

func (f *fakeRepo) ConsumePasswordReset(_ context.Context, id int64) error {
	for _, r := range f.resets {
		if r.id == id {
			now := time.Now()
			r.consumedAt = &now
		}
	}
	return nil
}
func (f *fakeRepo) SetAvatar(_ context.Context, id int64, _ []byte, contentType string) error { f.admins[id].AvatarType = contentType; return nil }
func (f *fakeRepo) ClearAvatar(_ context.Context, id int64) error { f.admins[id].AvatarType = ""; return nil }
func (f *fakeRepo) GetAvatar(_ context.Context, id int64) ([]byte, string, error) { a, ok := f.admins[id]; if !ok { return nil, "", errNotFound }; return nil, a.AvatarType, nil }
func (f *fakeRepo) UpdateProfile(_ context.Context, id int64, firstName, lastName, email, phone string, isMale bool) (admin.Admin, error) { a, ok := f.admins[id]; if !ok { return admin.Admin{}, errNotFound }; a.FirstName, a.LastName, a.Email, a.PhoneNumber, a.IsMale = firstName, lastName, email, phone, isMale; return *a, nil }

type fakeSMS struct{ lastPhone, lastMsg string }

func (s *fakeSMS) Send(_ context.Context, phone, msg string) error {
	s.lastPhone, s.lastMsg = phone, msg
	return nil
}

func newSvc(t *testing.T) (*Service, *fakeRepo, *admin.Admin) {
	t.Helper()
	repo := newFakeRepo()
	h, err := bcrypt.GenerateFromPassword([]byte("Amir@Pass1999"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash seed password: %v", err)
	}
	a := &admin.Admin{
		ID: 1, FirstName: "Amir", LastName: "Admin", Email: "admin@hesab.local",
		PhoneNumber: "9370843199", PasswordHash: string(h),
	}
	repo.admins[1] = a
	cfg := config.Config{
		JWTSecret:        "test-secret",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  720 * time.Hour,
		PasswordResetTTL: 5 * time.Minute,
		TwoFAPendingTTL:  5 * time.Minute,
		TOTPIssuer:       "Hesab Admin",
	}
	svc := NewService(repo, token.New(cfg), &fakeSMS{}, func() string { return "123456" }, time.Now, cfg)
	return svc, repo, a
}

func TestLoginWithoutTwoFA(t *testing.T) {
	svc, _, _ := newSvc(t)
	r, err := svc.Login(context.Background(), "09370843199", "Amir@Pass1999") // non-canonical phone accepted
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if r.TwoFARequired {
		t.Fatal("2fa unexpectedly required")
	}
	if r.AccessToken == "" || r.RefreshToken == "" {
		t.Fatal("missing tokens")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, _, _ := newSvc(t)
	if _, err := svc.Login(context.Background(), "9370843199", "wrong"); !errors.Is(err, admin.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestTwoFALoginFlow(t *testing.T) {
	svc, _, a := newSvc(t)
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Hesab Admin", AccountName: a.PhoneNumber})
	if err != nil {
		t.Fatalf("totp generate: %v", err)
	}
	a.TOTPSecret = key.Secret()

	r, err := svc.Login(context.Background(), "9370843199", "Amir@Pass1999")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !r.TwoFARequired || r.PendingToken == "" {
		t.Fatal("expected 2fa pending token")
	}
	if r.AccessToken != "" || r.RefreshToken != "" {
		t.Fatal("must not issue session before 2fa is verified")
	}

	code, _ := totp.GenerateCode(key.Secret(), time.Now())
	tok, err := svc.LoginVerify2FA(context.Background(), r.PendingToken, code)
	if err != nil {
		t.Fatalf("verify 2fa: %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Fatal("missing tokens after 2fa")
	}
	if _, err := svc.LoginVerify2FA(context.Background(), r.PendingToken, "000000"); err == nil {
		t.Fatal("invalid 2fa code accepted")
	}
}

func TestRefreshRotation(t *testing.T) {
	svc, _, _ := newSvc(t)
	r, _ := svc.Login(context.Background(), "9370843199", "Amir@Pass1999")

	rotated, err := svc.Refresh(context.Background(), r.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if rotated.RefreshToken == r.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := svc.Refresh(context.Background(), r.RefreshToken); !errors.Is(err, admin.ErrRefreshInvalid) {
		t.Fatalf("replay of old token: want ErrRefreshInvalid, got %v", err)
	}
	if _, err := svc.Refresh(context.Background(), rotated.RefreshToken); err != nil {
		t.Fatalf("rotated token should still refresh: %v", err)
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	svc, _, _ := newSvc(t)
	r, _ := svc.Login(context.Background(), "9370843199", "Amir@Pass1999")
	if err := svc.Logout(context.Background(), r.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), r.RefreshToken); !errors.Is(err, admin.ErrRefreshInvalid) {
		t.Fatalf("want ErrRefreshInvalid after logout, got %v", err)
	}
}

func TestForgotThenResetPassword(t *testing.T) {
	svc, _, _ := newSvc(t)
	old, _ := svc.Login(context.Background(), "9370843199", "Amir@Pass1999")

	if err := svc.ForgotPassword(context.Background(), "9370843199"); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), "9370843199", "123456", "NewPass2026"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := svc.Login(context.Background(), "9370843199", "NewPass2026"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if _, err := svc.Login(context.Background(), "9370843199", "Amir@Pass1999"); !errors.Is(err, admin.ErrInvalidCredentials) {
		t.Fatal("old password still works after reset")
	}
	if _, err := svc.Refresh(context.Background(), old.RefreshToken); !errors.Is(err, admin.ErrRefreshInvalid) {
		t.Fatal("pre-reset session not revoked")
	}
}

func TestResetPasswordWrongCode(t *testing.T) {
	svc, _, _ := newSvc(t)
	if err := svc.ForgotPassword(context.Background(), "9370843199"); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), "9370843199", "999999", "NewPass2026"); !errors.Is(err, admin.ErrResetCodeInvalid) {
		t.Fatalf("want ErrResetCodeInvalid, got %v", err)
	}
}

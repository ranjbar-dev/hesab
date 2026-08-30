// ponytail: parallel to adminauth, kept separate on purpose.
package clientauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/config"
	"hesab/api/internal/domain/user"
)

type RefreshToken struct {
	ID, UserID int64
	Hash       string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}
type PasswordReset struct {
	ID, UserID int64
	CodeHash   string
	ExpiresAt  time.Time
}
type Repository interface {
	UserByPhone(context.Context, string) (user.User, error)
	UserByID(context.Context, int64) (user.User, error)
	UpdatePassword(context.Context, int64, string) error
	SetTOTPSecret(context.Context, int64, string) error
	InsertRefreshToken(context.Context, int64, string, time.Time) error
	RefreshTokenByHash(context.Context, string) (RefreshToken, error)
	RevokeRefreshToken(context.Context, string) error
	RevokeAllRefreshTokens(context.Context, int64) error
	InvalidatePasswordResets(context.Context, int64) error
	InsertPasswordReset(context.Context, int64, string, time.Time) error
	LatestPasswordReset(context.Context, int64) (PasswordReset, error)
	ConsumePasswordReset(context.Context, int64) error
}
type TokenIssuer interface {
	IssueAccess(int64) (string, int, error)
	IssuePending(int64) (string, error)
	ParseAccess(string) (int64, error)
	ParsePending(string) (int64, error)
}
type SMSSender interface {
	Send(context.Context, string, string) error
}
type CodeGenerator func() string
type Clock func() time.Time
type Service struct {
	repo      Repository
	tokens    TokenIssuer
	sms       SMSSender
	code      CodeGenerator
	now       Clock
	cfg       config.Config
	dummyHash []byte
}
type Tokens struct {
	AccessToken  string
	ExpiresIn    int
	RefreshToken string
}
type LoginResult struct {
	Tokens
	TwoFARequired bool
	PendingToken  string
	User          user.User
}

func NewService(r Repository, t TokenIssuer, s SMSSender, c CodeGenerator, now Clock, cfg config.Config) *Service {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-123"), bcrypt.DefaultCost)
	return &Service{r, t, s, c, now, cfg, h}
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (s *Service) issue(ctx context.Context, id int64) (Tokens, error) {
	access, expires, e := s.tokens.IssueAccess(id)
	if e != nil {
		return Tokens{}, e
	}
	raw := make([]byte, 32)
	if _, e = rand.Read(raw); e != nil {
		return Tokens{}, e
	}
	r := hex.EncodeToString(raw)
	if e = s.repo.InsertRefreshToken(ctx, id, hash(r), s.now().Add(s.cfg.RefreshTokenTTL)); e != nil {
		return Tokens{}, e
	}
	return Tokens{access, expires, r}, nil
}
func (s *Service) Login(ctx context.Context, phone, password string) (LoginResult, error) {
	p, e := user.NormalizePhone(phone)
	if e != nil {
		return LoginResult{}, user.ErrInvalidCredentials
	}
	u, e := s.repo.UserByPhone(ctx, p)
	if e != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return LoginResult{}, user.ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, user.ErrInvalidCredentials
	}
	if u.TwoFAEnabled() {
		pending, e := s.tokens.IssuePending(u.ID)
		return LoginResult{TwoFARequired: true, PendingToken: pending, User: u}, e
	}
	t, e := s.issue(ctx, u.ID)
	return LoginResult{Tokens: t, User: u}, e
}
func (s *Service) LoginVerify2FA(ctx context.Context, pending, code string) (Tokens, error) {
	id, e := s.tokens.ParsePending(pending)
	if e != nil {
		return Tokens{}, user.ErrTwoFACodeInvalid
	}
	u, e := s.repo.UserByID(ctx, id)
	if e != nil || !totp.Validate(code, u.TOTPSecret) {
		return Tokens{}, user.ErrTwoFACodeInvalid
	}
	return s.issue(ctx, id)
}
func (s *Service) Refresh(ctx context.Context, raw string) (Tokens, error) {
	r, e := s.repo.RefreshTokenByHash(ctx, hash(raw))
	if e != nil || r.RevokedAt != nil || !r.ExpiresAt.After(s.now()) {
		return Tokens{}, user.ErrRefreshInvalid
	}
	if e = s.repo.RevokeRefreshToken(ctx, r.Hash); e != nil {
		return Tokens{}, user.ErrRefreshInvalid
	}
	return s.issue(ctx, r.UserID)
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, hash(raw))
}
func (s *Service) ForgotPassword(ctx context.Context, phone string) error {
	p, e := user.NormalizePhone(phone)
	if e != nil {
		return e
	}
	u, e := s.repo.UserByPhone(ctx, p)
	if e != nil {
		return nil
	}
	if e = s.repo.InvalidatePasswordResets(ctx, u.ID); e != nil {
		return e
	}
	c := s.code()
	if e = s.repo.InsertPasswordReset(ctx, u.ID, hash(c), s.now().Add(s.cfg.PasswordResetTTL)); e != nil {
		return e
	}
	return s.sms.Send(ctx, p, c+" کد بازیابی رمز عبور شماست")
}
func (s *Service) ResetPassword(ctx context.Context, phone, code, password string) error {
	if e := user.ValidatePassword(password); e != nil {
		return e
	}
	p, e := user.NormalizePhone(phone)
	if e != nil {
		return user.ErrResetCodeInvalid
	}
	u, e := s.repo.UserByPhone(ctx, p)
	if e != nil {
		return user.ErrResetCodeInvalid
	}
	r, e := s.repo.LatestPasswordReset(ctx, u.ID)
	if e != nil || r.CodeHash != hash(code) {
		return user.ErrResetCodeInvalid
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	if e = s.repo.ConsumePasswordReset(ctx, r.ID); e != nil {
		return e
	}
	if e = s.repo.UpdatePassword(ctx, u.ID, string(h)); e != nil {
		return e
	}
	return s.repo.RevokeAllRefreshTokens(ctx, u.ID)
}
func (s *Service) Setup2FA(ctx context.Context, id int64) (string, string, error) {
	u, e := s.repo.UserByID(ctx, id)
	if e != nil {
		return "", "", e
	}
	k, e := totp.Generate(totp.GenerateOpts{Issuer: s.cfg.ClientTOTPIssuer, AccountName: u.PhoneNumber})
	if e != nil {
		return "", "", e
	}
	return k.Secret(), k.URL(), nil
}
func (s *Service) Activate2FA(ctx context.Context, id int64, secret, code string) error {
	if !totp.Validate(code, secret) {
		return user.ErrTwoFACodeInvalid
	}
	return s.repo.SetTOTPSecret(ctx, id, secret)
}
func (s *Service) Disable2FA(ctx context.Context, id int64, password string) error {
	u, e := s.repo.UserByID(ctx, id)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return user.ErrInvalidCredentials
	}
	return s.repo.SetTOTPSecret(ctx, id, "")
}
func (s *Service) Me(ctx context.Context, id int64) (user.User, error) {
	return s.repo.UserByID(ctx, id)
}

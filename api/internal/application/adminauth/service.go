package adminauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/config"
	"hesab/api/internal/domain/admin"
	"time"
)

type RefreshToken struct {
	ID, AdminID int64
	Hash        string
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}
type PasswordReset struct {
	ID, AdminID int64
	CodeHash    string
	ExpiresAt   time.Time
}
type Repository interface {
	AdminByPhone(context.Context, string) (admin.Admin, error)
	AdminByID(context.Context, int64) (admin.Admin, error)
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
	Admin         admin.Admin
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
	p, e := admin.NormalizePhone(phone)
	if e != nil {
		return LoginResult{}, admin.ErrInvalidCredentials
	}
	a, e := s.repo.AdminByPhone(ctx, p)
	if e != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return LoginResult{}, admin.ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, admin.ErrInvalidCredentials
	}
	if a.TwoFAEnabled() {
		pending, e := s.tokens.IssuePending(a.ID)
		return LoginResult{TwoFARequired: true, PendingToken: pending, Admin: a}, e
	}
	t, e := s.issue(ctx, a.ID)
	return LoginResult{Tokens: t, Admin: a}, e
}
func (s *Service) LoginVerify2FA(ctx context.Context, pending, code string) (Tokens, error) {
	id, e := s.tokens.ParsePending(pending)
	if e != nil {
		return Tokens{}, admin.ErrTwoFACodeInvalid
	}
	a, e := s.repo.AdminByID(ctx, id)
	if e != nil || !totp.Validate(code, a.TOTPSecret) {
		return Tokens{}, admin.ErrTwoFACodeInvalid
	}
	return s.issue(ctx, id)
}
func (s *Service) Refresh(ctx context.Context, raw string) (Tokens, error) {
	r, e := s.repo.RefreshTokenByHash(ctx, hash(raw))
	if e != nil || r.RevokedAt != nil || !r.ExpiresAt.After(s.now()) {
		return Tokens{}, admin.ErrRefreshInvalid
	}
	if e = s.repo.RevokeRefreshToken(ctx, r.Hash); e != nil {
		return Tokens{}, admin.ErrRefreshInvalid
	}
	return s.issue(ctx, r.AdminID)
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, hash(raw))
}
func (s *Service) ForgotPassword(ctx context.Context, phone string) error {
	p, e := admin.NormalizePhone(phone)
	if e != nil {
		return e
	}
	a, e := s.repo.AdminByPhone(ctx, p)
	if e != nil {
		return nil
	}
	if e = s.repo.InvalidatePasswordResets(ctx, a.ID); e != nil {
		return e
	}
	c := s.code()
	if e = s.repo.InsertPasswordReset(ctx, a.ID, hash(c), s.now().Add(s.cfg.PasswordResetTTL)); e != nil {
		return e
	}
	return s.sms.Send(ctx, p, c+" کد بازیابی رمز عبور شماست")
}
func (s *Service) ResetPassword(ctx context.Context, phone, code, password string) error {
	if e := admin.ValidatePassword(password); e != nil {
		return e
	}
	p, e := admin.NormalizePhone(phone)
	if e != nil {
		return admin.ErrResetCodeInvalid
	}
	a, e := s.repo.AdminByPhone(ctx, p)
	if e != nil {
		return admin.ErrResetCodeInvalid
	}
	r, e := s.repo.LatestPasswordReset(ctx, a.ID)
	if e != nil || r.CodeHash != hash(code) {
		return admin.ErrResetCodeInvalid
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	if e = s.repo.ConsumePasswordReset(ctx, r.ID); e != nil {
		return e
	}
	if e = s.repo.UpdatePassword(ctx, a.ID, string(h)); e != nil {
		return e
	}
	return s.repo.RevokeAllRefreshTokens(ctx, a.ID)
}
func (s *Service) Setup2FA(ctx context.Context, id int64) (string, string, error) {
	a, e := s.repo.AdminByID(ctx, id)
	if e != nil {
		return "", "", e
	}
	k, e := totp.Generate(totp.GenerateOpts{Issuer: s.cfg.TOTPIssuer, AccountName: a.PhoneNumber})
	if e != nil {
		return "", "", e
	}
	return k.Secret(), k.URL(), nil
}
func (s *Service) Activate2FA(ctx context.Context, id int64, secret, code string) error {
	if !totp.Validate(code, secret) {
		return admin.ErrTwoFACodeInvalid
	}
	return s.repo.SetTOTPSecret(ctx, id, secret)
}
func (s *Service) Disable2FA(ctx context.Context, id int64, password string) error {
	a, e := s.repo.AdminByID(ctx, id)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) != nil {
		return admin.ErrInvalidCredentials
	}
	return s.repo.SetTOTPSecret(ctx, id, "")
}
func (s *Service) Me(ctx context.Context, id int64) (admin.Admin, error) {
	return s.repo.AdminByID(ctx, id)
}

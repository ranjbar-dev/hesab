package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"hesab/api/internal/application/adminauth"
	"hesab/api/internal/domain/admin"
	"hesab/api/internal/infrastructure/db/sqlc"
	"time"
)

type AdminRepo struct{ q *sqlc.Queries }

func NewAdminRepo(q *sqlc.Queries) *AdminRepo { return &AdminRepo{q} }
func adminFrom(a sqlc.Admin) admin.Admin {
	return admin.Admin{ID: a.ID, FirstName: a.FirstName, LastName: a.LastName, Email: a.Email, PhoneNumber: a.PhoneNumber, IsMale: a.IsMale, PasswordHash: a.PasswordHash, TOTPSecret: a.TotpSecret, AvatarType: a.AvatarType, CreatedAt: a.CreatedAt.Time}
}
func (r *AdminRepo) AdminByPhone(c context.Context, p string) (admin.Admin, error) {
	a, e := r.q.GetAdminByPhone(c, p)
	if errors.Is(e, pgx.ErrNoRows) {
		return admin.Admin{}, admin.ErrInvalidCredentials
	}
	return adminFrom(a), e
}
func (r *AdminRepo) AdminByID(c context.Context, id int64) (admin.Admin, error) {
	a, e := r.q.GetAdminByID(c, id)
	return adminFrom(a), e
}
func (r *AdminRepo) UpdatePassword(c context.Context, id int64, h string) error {
	return r.q.UpdateAdminPassword(c, sqlc.UpdateAdminPasswordParams{ID: id, PasswordHash: h})
}
func (r *AdminRepo) SetTOTPSecret(c context.Context, id int64, s string) error {
	return r.q.SetAdminTOTPSecret(c, sqlc.SetAdminTOTPSecretParams{ID: id, TotpSecret: s})
}
func (r *AdminRepo) InsertRefreshToken(c context.Context, id int64, h string, e time.Time) error {
	_, x := r.q.InsertRefreshToken(c, sqlc.InsertRefreshTokenParams{AdminID: id, TokenHash: h, ExpiresAt: pgtype.Timestamptz{Time: e, Valid: true}})
	return x
}
func (r *AdminRepo) RefreshTokenByHash(c context.Context, h string) (adminauth.RefreshToken, error) {
	v, e := r.q.GetRefreshToken(c, h)
	if errors.Is(e, pgx.ErrNoRows) {
		return adminauth.RefreshToken{}, admin.ErrRefreshInvalid
	}
	var revoked *time.Time
	if v.RevokedAt.Valid {
		revoked = &v.RevokedAt.Time
	}
	return adminauth.RefreshToken{ID: v.ID, AdminID: v.AdminID, Hash: v.TokenHash, ExpiresAt: v.ExpiresAt.Time, RevokedAt: revoked}, e
}
func (r *AdminRepo) RevokeRefreshToken(c context.Context, h string) error {
	return r.q.RevokeRefreshToken(c, h)
}
func (r *AdminRepo) RevokeAllRefreshTokens(c context.Context, id int64) error {
	return r.q.RevokeAllAdminRefreshTokens(c, id)
}
func (r *AdminRepo) InvalidatePasswordResets(c context.Context, id int64) error {
	return r.q.InvalidateAdminPasswordResets(c, id)
}
func (r *AdminRepo) InsertPasswordReset(c context.Context, id int64, h string, e time.Time) error {
	_, x := r.q.InsertPasswordReset(c, sqlc.InsertPasswordResetParams{AdminID: id, CodeHash: h, ExpiresAt: pgtype.Timestamptz{Time: e, Valid: true}})
	return x
}
func (r *AdminRepo) LatestPasswordReset(c context.Context, id int64) (adminauth.PasswordReset, error) {
	v, e := r.q.GetLatestPasswordReset(c, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return adminauth.PasswordReset{}, admin.ErrResetCodeInvalid
	}
	return adminauth.PasswordReset{ID: v.ID, AdminID: v.AdminID, CodeHash: v.CodeHash, ExpiresAt: v.ExpiresAt.Time}, e
}
func (r *AdminRepo) ConsumePasswordReset(c context.Context, id int64) error {
	return r.q.ConsumePasswordReset(c, id)
}
func (r *AdminRepo) SetAvatar(c context.Context, id int64, data []byte, contentType string) error {
	return r.q.SetAdminAvatar(c, sqlc.SetAdminAvatarParams{ID: id, Avatar: data, AvatarType: contentType})
}
func (r *AdminRepo) ClearAvatar(c context.Context, id int64) error {
	return r.q.ClearAdminAvatar(c, id)
}
func (r *AdminRepo) GetAvatar(c context.Context, id int64) ([]byte, string, error) {
	v, e := r.q.GetAdminAvatar(c, id)
	return v.Avatar, v.AvatarType, e
}
func (r *AdminRepo) UpdateProfile(c context.Context, id int64, firstName, lastName, email, phone string, isMale bool) (admin.Admin, error) {
	v, e := r.q.UpdateAdminProfile(c, sqlc.UpdateAdminProfileParams{ID: id, FirstName: firstName, LastName: lastName, Email: email, PhoneNumber: phone, IsMale: isMale})
	return adminFrom(v), e
}

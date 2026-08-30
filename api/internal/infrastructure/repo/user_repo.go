// ponytail: parallel to adminauth, kept separate on purpose.
package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"hesab/api/internal/application/clientauth"
	"hesab/api/internal/domain/user"
	"hesab/api/internal/infrastructure/db/sqlc"
	"time"
)

type UserRepo struct{ q *sqlc.Queries }

func NewUserRepo(q *sqlc.Queries) *UserRepo { return &UserRepo{q} }
func userFrom(u sqlc.User) user.User {
	return user.User{ID: u.ID, FirstName: u.FirstName, LastName: u.LastName, Email: u.Email, PhoneNumber: u.PhoneNumber, PasswordHash: u.PasswordHash, TOTPSecret: u.TotpSecret, CreatedAt: u.CreatedAt.Time}
}
func (r *UserRepo) UserByPhone(c context.Context, p string) (user.User, error) {
	u, e := r.q.GetUserByPhone(c, p)
	if errors.Is(e, pgx.ErrNoRows) {
		return user.User{}, user.ErrInvalidCredentials
	}
	return userFrom(u), e
}
func (r *UserRepo) UserByID(c context.Context, id int64) (user.User, error) {
	u, e := r.q.GetUserByID(c, id)
	return userFrom(u), e
}
func (r *UserRepo) UpdatePassword(c context.Context, id int64, h string) error {
	return r.q.UpdateUserPassword(c, sqlc.UpdateUserPasswordParams{ID: id, PasswordHash: h})
}
func (r *UserRepo) SetTOTPSecret(c context.Context, id int64, s string) error {
	return r.q.SetUserTOTPSecret(c, sqlc.SetUserTOTPSecretParams{ID: id, TotpSecret: s})
}
func (r *UserRepo) InsertRefreshToken(c context.Context, id int64, h string, e time.Time) error {
	_, x := r.q.InsertUserRefreshToken(c, sqlc.InsertUserRefreshTokenParams{UserID: id, TokenHash: h, ExpiresAt: pgtype.Timestamptz{Time: e, Valid: true}})
	return x
}
func (r *UserRepo) RefreshTokenByHash(c context.Context, h string) (clientauth.RefreshToken, error) {
	v, e := r.q.GetUserRefreshToken(c, h)
	if errors.Is(e, pgx.ErrNoRows) {
		return clientauth.RefreshToken{}, user.ErrRefreshInvalid
	}
	var revoked *time.Time
	if v.RevokedAt.Valid {
		revoked = &v.RevokedAt.Time
	}
	return clientauth.RefreshToken{ID: v.ID, UserID: v.UserID, Hash: v.TokenHash, ExpiresAt: v.ExpiresAt.Time, RevokedAt: revoked}, e
}
func (r *UserRepo) RevokeRefreshToken(c context.Context, h string) error {
	return r.q.RevokeUserRefreshToken(c, h)
}
func (r *UserRepo) RevokeAllRefreshTokens(c context.Context, id int64) error {
	return r.q.RevokeAllUserRefreshTokens(c, id)
}
func (r *UserRepo) InvalidatePasswordResets(c context.Context, id int64) error {
	return r.q.InvalidateUserPasswordResets(c, id)
}
func (r *UserRepo) InsertPasswordReset(c context.Context, id int64, h string, e time.Time) error {
	_, x := r.q.InsertUserPasswordReset(c, sqlc.InsertUserPasswordResetParams{UserID: id, CodeHash: h, ExpiresAt: pgtype.Timestamptz{Time: e, Valid: true}})
	return x
}
func (r *UserRepo) LatestPasswordReset(c context.Context, id int64) (clientauth.PasswordReset, error) {
	v, e := r.q.GetLatestUserPasswordReset(c, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return clientauth.PasswordReset{}, user.ErrResetCodeInvalid
	}
	return clientauth.PasswordReset{ID: v.ID, UserID: v.UserID, CodeHash: v.CodeHash, ExpiresAt: v.ExpiresAt.Time}, e
}
func (r *UserRepo) ConsumePasswordReset(c context.Context, id int64) error {
	return r.q.ConsumeUserPasswordReset(c, id)
}

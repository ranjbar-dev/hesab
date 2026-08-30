package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"hesab/api/internal/application/usersadmin"
	"hesab/api/internal/domain/user"
	"hesab/api/internal/infrastructure/db/sqlc"
)

type UserAdminRepo struct{ q *sqlc.Queries }

func NewUserAdminRepo(q *sqlc.Queries) *UserAdminRepo { return &UserAdminRepo{q} }

func userAdminRow(v sqlc.User) user.User {
	var nationalID string
	if v.NationalID.Valid {
		nationalID = v.NationalID.String
	}
	var deletedAt *time.Time
	if v.DeletedAt.Valid {
		deletedAt = &v.DeletedAt.Time
	}
	return user.User{ID: v.ID, FirstName: v.FirstName, LastName: v.LastName, Email: v.Email, PhoneNumber: v.PhoneNumber, PasswordHash: v.PasswordHash, TOTPSecret: v.TotpSecret, NationalID: nationalID, AccountType: v.AccountType, Status: v.Status, DeletedAt: deletedAt, CreatedAt: v.CreatedAt.Time}
}
func nullableString(v string) pgtype.Text {
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}
func nullableTime(v *time.Time) pgtype.Timestamptz {
	if v == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *v, Valid: true}
}
func noRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return user.ErrUserNotFound
	}
	return err
}

func (r *UserAdminRepo) Create(c context.Context, in usersadmin.NewUser) (user.User, error) {
	v, err := r.q.AdminCreateUser(c, sqlc.AdminCreateUserParams{FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, PhoneNumber: in.PhoneNumber, NationalID: nullableString(in.NationalID), AccountType: in.AccountType, PasswordHash: in.PasswordHash})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return user.User{}, user.ErrPhoneTaken
	}
	if err != nil {
		return user.User{}, err
	}
	return userAdminRow(v), nil
}
func (r *UserAdminRepo) Get(c context.Context, id int64) (user.User, error) {
	v, e := r.q.AdminGetUser(c, id)
	if e != nil {
		return user.User{}, noRows(e)
	}
	return userAdminRow(v), nil
}
func (r *UserAdminRepo) listParams(f usersadmin.ListFilter) sqlc.AdminListUsersParams {
	return sqlc.AdminListUsersParams{FirstName: nullableString(f.FirstName), LastName: nullableString(f.LastName), Phone: nullableString(f.Phone), Status: nullableString(f.Status), CreatedFrom: nullableTime(f.CreatedFrom), CreatedTo: nullableTime(f.CreatedTo), Lim: f.Limit, Off: f.Offset}
}
func (r *UserAdminRepo) List(c context.Context, f usersadmin.ListFilter) ([]user.User, error) {
	vs, e := r.q.AdminListUsers(c, r.listParams(f))
	if e != nil {
		return nil, e
	}
	out := make([]user.User, len(vs))
	for i, v := range vs {
		out[i] = userAdminRow(v)
	}
	return out, nil
}
func (r *UserAdminRepo) Count(c context.Context, f usersadmin.ListFilter) (int64, error) {
	p := r.listParams(f)
	return r.q.AdminCountUsers(c, sqlc.AdminCountUsersParams{FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone, Status: p.Status, CreatedFrom: p.CreatedFrom, CreatedTo: p.CreatedTo})
}
func (r *UserAdminRepo) UpdateProfile(c context.Context, id int64, in usersadmin.Profile) (user.User, error) {
	v, e := r.q.AdminUpdateUserProfile(c, sqlc.AdminUpdateUserProfileParams{ID: id, FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, NationalID: nullableString(in.NationalID), AccountType: in.AccountType})
	if e != nil {
		return user.User{}, noRows(e)
	}
	return userAdminRow(v), nil
}
func (r *UserAdminRepo) SetStatus(c context.Context, id int64, status string) (user.User, error) {
	v, e := r.q.AdminSetUserStatus(c, sqlc.AdminSetUserStatusParams{ID: id, Status: status})
	if e != nil {
		return user.User{}, noRows(e)
	}
	return userAdminRow(v), nil
}
func (r *UserAdminRepo) SetPassword(c context.Context, id int64, hash string) error {
	return r.q.AdminSetUserPassword(c, sqlc.AdminSetUserPasswordParams{ID: id, PasswordHash: hash})
}
func (r *UserAdminRepo) SoftDelete(c context.Context, id int64) error {
	return r.q.AdminSoftDeleteUser(c, id)
}
func (r *UserAdminRepo) RevokeAllSessions(c context.Context, id int64) error {
	return r.q.RevokeAllUserRefreshTokens(c, id)
}

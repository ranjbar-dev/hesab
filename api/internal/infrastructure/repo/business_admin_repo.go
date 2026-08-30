package repo

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	app "hesab/api/internal/application/businessadmin"
	"hesab/api/internal/domain/business"
	"hesab/api/internal/domain/user"
	"hesab/api/internal/infrastructure/db/sqlc"
)

type BusinessAdminRepo struct{ q *sqlc.Queries }

func NewBusinessAdminRepo(q *sqlc.Queries) *BusinessAdminRepo { return &BusinessAdminRepo{q} }
func nameFilter(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
func owner(id int64, f, l, p string) app.Owner {
	return app.Owner{ID: id, FirstName: f, LastName: l, PhoneNumber: p}
}
func (r *BusinessAdminRepo) List(c context.Context, n string, lim, off int32) ([]app.ListRow, error) {
	v, e := r.q.AdminListBusinesses(c, sqlc.AdminListBusinessesParams{Name: nameFilter(n), Lim: lim, Off: off})
	if e != nil {
		return nil, e
	}
	o := make([]app.ListRow, len(v))
	for i, x := range v {
		o[i] = app.ListRow{Business: business.Business{ID: x.ID, Name: x.Name, OwnerUserID: x.OwnerUserID, CreatedAt: x.CreatedAt.Time}, Owner: owner(x.OwnerUserID, x.OwnerFirstName, x.OwnerLastName, x.OwnerPhoneNumber), MemberCount: x.MemberCount}
	}
	return o, nil
}
func (r *BusinessAdminRepo) Count(c context.Context, n string) (int64, error) {
	return r.q.AdminCountBusinesses(c, nameFilter(n))
}
func (r *BusinessAdminRepo) Get(c context.Context, id int64) (business.Business, app.Owner, error) {
	v, e := r.q.AdminGetBusiness(c, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return business.Business{}, app.Owner{}, business.ErrNotFound
	}
	return business.Business{ID: v.ID, Name: v.Name, OwnerUserID: v.OwnerUserID, CreatedAt: v.CreatedAt.Time}, owner(v.OwnerUserID, v.OwnerFirstName, v.OwnerLastName, v.OwnerPhoneNumber), e
}
func (r *BusinessAdminRepo) Members(c context.Context, id int64) ([]business.Member, error) {
	return (&BusinessRepo{r.q}).Members(c, id)
}
func (r *BusinessAdminRepo) GetUser(c context.Context, id int64) (user.User, error) {
	v, e := r.q.GetUserByID(c, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return user.User{}, user.ErrUserNotFound
	}
	return activeUser(v), e
}
func (r *BusinessAdminRepo) Create(c context.Context, n string, id int64) (business.Business, error) {
	return (&BusinessRepo{r.q}).Create(c, n, id)
}
func (r *BusinessAdminRepo) AddMember(c context.Context, b, u int64, role string) error {
	return (&BusinessRepo{r.q}).AddMember(c, b, u, role)
}
func (r *BusinessAdminRepo) Rename(c context.Context, id int64, n string) (business.Business, error) {
	return (&BusinessRepo{r.q}).Rename(c, id, n)
}
func (r *BusinessAdminRepo) SoftDelete(c context.Context, id int64) error {
	return (&BusinessRepo{r.q}).SoftDelete(c, id)
}
func (r *BusinessAdminRepo) MemberRole(c context.Context, b, u int64) (string, error) {
	return (&BusinessRepo{r.q}).MemberRole(c, b, u)
}
func (r *BusinessAdminRepo) UpdateRole(c context.Context, b, u int64, role string) error {
	return (&BusinessRepo{r.q}).UpdateRole(c, b, u, role)
}
func (r *BusinessAdminRepo) RemoveMember(c context.Context, b, u int64) error {
	return (&BusinessRepo{r.q}).RemoveMember(c, b, u)
}
func (r *BusinessAdminRepo) ActiveUserByPhone(c context.Context, p string) (user.User, error) {
	return (&BusinessRepo{r.q}).ActiveUserByPhone(c, p)
}
func (r *BusinessAdminRepo) Owned(c context.Context, id int64) ([]app.OwnedRow, error) {
	v, e := r.q.AdminListOwnedBusinesses(c, id)
	if e != nil {
		return nil, e
	}
	o := make([]app.OwnedRow, len(v))
	for i, x := range v {
		o[i] = app.OwnedRow{ID: x.ID, Name: x.Name, MemberCount: x.MemberCount, CreatedAt: x.CreatedAt.Time}
	}
	return o, nil
}
func (r *BusinessAdminRepo) Joined(c context.Context, id int64) ([]app.JoinedRow, error) {
	v, e := r.q.AdminListJoinedBusinesses(c, id)
	if e != nil {
		return nil, e
	}
	o := make([]app.JoinedRow, len(v))
	for i, x := range v {
		o[i] = app.JoinedRow{ID: x.ID, Name: x.Name, Role: x.Role, OwnerName: x.OwnerName, CreatedAt: x.CreatedAt.Time}
	}
	return o, nil
}

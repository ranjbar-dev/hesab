package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	app "hesab/api/internal/application/business"
	"hesab/api/internal/domain/business"
	"hesab/api/internal/domain/user"
	"hesab/api/internal/infrastructure/db/sqlc"
	"time"
)

type BusinessRepo struct{ q *sqlc.Queries }

func NewBusinessRepo(q *sqlc.Queries) *BusinessRepo { return &BusinessRepo{q} }
func businessRow(v sqlc.Business) business.Business {
	var d *time.Time
	if v.DeletedAt.Valid {
		d = &v.DeletedAt.Time
	}
	return business.Business{ID: v.ID, Name: v.Name, OwnerUserID: v.OwnerUserID, CreatedAt: v.CreatedAt.Time, DeletedAt: d}
}
func activeUser(v sqlc.User) user.User {
	return user.User{ID: v.ID, FirstName: v.FirstName, LastName: v.LastName, PhoneNumber: v.PhoneNumber, CreatedAt: v.CreatedAt.Time}
}
func bizNoRows(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return business.ErrNotFound
	}
	return e
}
func memberNoRows(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return business.ErrNotMember
	}
	return e
}
func (r *BusinessRepo) Create(c context.Context, n string, uid int64) (business.Business, error) {
	v, e := r.q.CreateBusiness(c, sqlc.CreateBusinessParams{Name: n, OwnerUserID: uid})
	return businessRow(v), e
}
func (r *BusinessRepo) AddMember(c context.Context, bid, uid int64, role string) error {
	_, e := r.q.AddMember(c, sqlc.AddMemberParams{BusinessID: bid, UserID: uid, Role: role})
	var p *pgconn.PgError
	if errors.As(e, &p) && p.Code == "23505" {
		return business.ErrAlreadyMember
	}
	return e
}
func (r *BusinessRepo) GetBusiness(c context.Context, id int64) (business.Business, error) {
	v, e := r.q.GetBusiness(c, id)
	return businessRow(v), bizNoRows(e)
}
func (r *BusinessRepo) MemberRole(c context.Context, bid, uid int64) (string, error) {
	v, e := r.q.GetMemberRole(c, sqlc.GetMemberRoleParams{BusinessID: bid, UserID: uid})
	return v, memberNoRows(e)
}
func (r *BusinessRepo) List(c context.Context, uid int64) ([]app.BusinessWithRole, error) {
	v, e := r.q.ListUserBusinesses(c, uid)
	if e != nil {
		return nil, e
	}
	o := make([]app.BusinessWithRole, len(v))
	for i, x := range v {
		o[i] = app.BusinessWithRole{Business: business.Business{ID: x.ID, Name: x.Name, OwnerUserID: x.OwnerUserID, CreatedAt: x.CreatedAt.Time}, Role: x.Role}
	}
	return o, nil
}
func (r *BusinessRepo) Members(c context.Context, bid int64) ([]business.Member, error) {
	v, e := r.q.ListMembers(c, bid)
	if e != nil {
		return nil, e
	}
	o := make([]business.Member, len(v))
	for i, x := range v {
		o[i] = business.Member{UserID: x.UserID, Role: x.Role, FirstName: x.FirstName, LastName: x.LastName, PhoneNumber: x.PhoneNumber, CreatedAt: x.CreatedAt.Time}
	}
	return o, nil
}
func (r *BusinessRepo) Rename(c context.Context, id int64, n string) (business.Business, error) {
	v, e := r.q.RenameBusiness(c, sqlc.RenameBusinessParams{ID: id, Name: n})
	return businessRow(v), bizNoRows(e)
}
func (r *BusinessRepo) SoftDelete(c context.Context, id int64) error {
	if _, e := r.GetBusiness(c, id); e != nil {
		return e
	}
	return r.q.SoftDeleteBusiness(c, id)
}
func (r *BusinessRepo) UpdateRole(c context.Context, bid, uid int64, role string) error {
	_, e := r.q.UpdateMemberRole(c, sqlc.UpdateMemberRoleParams{BusinessID: bid, UserID: uid, Role: role})
	return memberNoRows(e)
}
func (r *BusinessRepo) RemoveMember(c context.Context, bid, uid int64) error {
	return r.q.RemoveMember(c, sqlc.RemoveMemberParams{BusinessID: bid, UserID: uid})
}
func (r *BusinessRepo) ActiveUserByPhone(c context.Context, p string) (user.User, error) {
	v, e := r.q.GetActiveUserByPhone(c, p)
	if errors.Is(e, pgx.ErrNoRows) {
		return user.User{}, business.ErrInviteeNotRegistered
	}
	return activeUser(v), e
}
func (r *BusinessRepo) CreateInvite(c context.Context, bid, uid int64, role string, by int64) error {
	_, e := r.q.CreateInvite(c, sqlc.CreateInviteParams{BusinessID: bid, UserID: uid, Role: role, InvitedBy: pgtype.Int8{Int64: by, Valid: true}})
	var p *pgconn.PgError
	if errors.As(e, &p) && p.Code == "23505" {
		return business.ErrInvitePending
	}
	return e
}
func (r *BusinessRepo) PendingForUser(c context.Context, uid int64) ([]business.Invite, error) {
	v, e := r.q.ListPendingInvitesForUser(c, uid)
	if e != nil {
		return nil, e
	}
	o := make([]business.Invite, len(v))
	for i, x := range v {
		o[i] = business.Invite{ID: x.ID, BusinessID: x.BusinessID, BusinessName: x.BusinessName, Role: x.Role, Status: x.Status, InvitedByName: fmt.Sprint(x.InvitedByName), CreatedAt: x.CreatedAt.Time}
	}
	return o, nil
}
func (r *BusinessRepo) PendingForBusiness(c context.Context, bid int64) ([]business.Invite, error) {
	v, e := r.q.ListPendingInvitesForBusiness(c, bid)
	if e != nil {
		return nil, e
	}
	o := make([]business.Invite, len(v))
	for i, x := range v {
		o[i] = business.Invite{ID: x.ID, BusinessID: x.BusinessID, UserID: x.UserID, FirstName: x.FirstName, LastName: x.LastName, PhoneNumber: x.PhoneNumber, Role: x.Role, CreatedAt: x.CreatedAt.Time}
	}
	return o, nil
}
func (r *BusinessRepo) Invite(c context.Context, id int64) (business.Invite, error) {
	v, e := r.q.GetInvite(c, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return business.Invite{}, business.ErrInviteNotFound
	}
	return business.Invite{ID: v.ID, BusinessID: v.BusinessID, UserID: v.UserID, Role: v.Role, Status: v.Status, CreatedAt: v.CreatedAt.Time}, e
}
func (r *BusinessRepo) SetInviteStatus(c context.Context, id int64, s string) error {
	return r.q.SetInviteStatus(c, sqlc.SetInviteStatusParams{ID: id, Status: s})
}

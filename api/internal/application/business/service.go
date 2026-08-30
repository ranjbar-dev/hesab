package business

import (
	"context"
	"hesab/api/internal/domain/business"
	"hesab/api/internal/domain/user"
	"strings"
)

type BusinessWithRole struct {
	business.Business
	Role string
}
type Repository interface {
	Create(context.Context, string, int64) (business.Business, error)
	AddMember(context.Context, int64, int64, string) error
	GetBusiness(context.Context, int64) (business.Business, error)
	MemberRole(context.Context, int64, int64) (string, error)
	List(context.Context, int64) ([]BusinessWithRole, error)
	Members(context.Context, int64) ([]business.Member, error)
	Rename(context.Context, int64, string) (business.Business, error)
	SoftDelete(context.Context, int64) error
	UpdateRole(context.Context, int64, int64, string) error
	RemoveMember(context.Context, int64, int64) error
	ActiveUserByPhone(context.Context, string) (user.User, error)
	CreateInvite(context.Context, int64, int64, string, int64) error
	PendingForUser(context.Context, int64) ([]business.Invite, error)
	PendingForBusiness(context.Context, int64) ([]business.Invite, error)
	Invite(context.Context, int64) (business.Invite, error)
	SetInviteStatus(context.Context, int64, string) error
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{r} }
func name(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", business.ErrNameRequired
	}
	return v, nil
}
func (s *Service) role(c context.Context, uid, bid int64) (string, error) {
	return s.repo.MemberRole(c, bid, uid)
}
func (s *Service) managed(c context.Context, uid, bid int64) (string, error) {
	r, e := s.role(c, uid, bid)
	if e != nil {
		return "", e
	}
	if !business.CanManageMembers(r) {
		return "", business.ErrForbidden
	}
	return r, nil
}
func (s *Service) List(c context.Context, uid int64) ([]BusinessWithRole, error) {
	return s.repo.List(c, uid)
}
func (s *Service) Create(c context.Context, uid int64, n string) (business.Business, error) {
	n, e := name(n)
	if e != nil {
		return business.Business{}, e
	}
	b, e := s.repo.Create(c, n, uid)
	if e != nil {
		return b, e
	}
	if e = s.repo.AddMember(c, b.ID, uid, business.RoleOwner); e != nil {
		return business.Business{}, e
	}
	return b, nil
}
func (s *Service) Get(c context.Context, uid, bid int64) (business.Business, string, error) {
	r, e := s.role(c, uid, bid)
	if e != nil {
		return business.Business{}, "", e
	}
	b, e := s.repo.GetBusiness(c, bid)
	return b, r, e
}
func (s *Service) Rename(c context.Context, uid, bid int64, n string) (business.Business, error) {
	if _, e := s.managed(c, uid, bid); e != nil {
		return business.Business{}, e
	}
	n, e := name(n)
	if e != nil {
		return business.Business{}, e
	}
	return s.repo.Rename(c, bid, n)
}
func (s *Service) Delete(c context.Context, uid, bid int64) error {
	r, e := s.role(c, uid, bid)
	if e != nil {
		return e
	}
	if r != business.RoleOwner {
		return business.ErrForbidden
	}
	return s.repo.SoftDelete(c, bid)
}
func (s *Service) Members(c context.Context, uid, bid int64) ([]business.Member, string, error) {
	r, e := s.role(c, uid, bid)
	if e != nil {
		return nil, "", e
	}
	v, e := s.repo.Members(c, bid)
	return v, r, e
}
func (s *Service) Invite(c context.Context, uid, bid int64, phone, role string) error {
	if _, e := s.managed(c, uid, bid); e != nil {
		return e
	}
	if !business.AssignableRole(role) {
		return business.ErrInvalidRole
	}
	p, e := user.NormalizePhone(phone)
	if e != nil {
		return e
	}
	u, e := s.repo.ActiveUserByPhone(c, p)
	if e != nil {
		return business.ErrInviteeNotRegistered
	}
	if _, e = s.repo.MemberRole(c, bid, u.ID); e == nil {
		return business.ErrAlreadyMember
	}
	return s.repo.CreateInvite(c, bid, u.ID, role, uid)
}
func (s *Service) CancelInvite(c context.Context, uid, bid, iid int64) error {
	if _, e := s.managed(c, uid, bid); e != nil {
		return e
	}
	i, e := s.repo.Invite(c, iid)
	if e != nil || i.BusinessID != bid || i.Status != "pending" {
		return business.ErrInviteNotFound
	}
	return s.repo.SetInviteStatus(c, iid, "cancelled")
}
func (s *Service) ChangeRole(c context.Context, uid, bid, target int64, role string) error {
	if _, e := s.managed(c, uid, bid); e != nil {
		return e
	}
	if !business.AssignableRole(role) {
		return business.ErrInvalidRole
	}
	r, e := s.repo.MemberRole(c, bid, target)
	if e != nil {
		return e
	}
	if r == business.RoleOwner {
		return business.ErrCannotTargetOwner
	}
	return s.repo.UpdateRole(c, bid, target, role)
}
func (s *Service) RemoveMember(c context.Context, uid, bid, target int64) error {
	r, e := s.role(c, uid, bid)
	if e != nil {
		return e
	}
	tr, e := s.repo.MemberRole(c, bid, target)
	if e != nil {
		return e
	}
	if target == uid {
		if r == business.RoleOwner {
			return business.ErrOwnerCannotLeave
		}
	} else if !business.CanManageMembers(r) {
		return business.ErrForbidden
	}
	if tr == business.RoleOwner {
		return business.ErrCannotTargetOwner
	}
	return s.repo.RemoveMember(c, bid, target)
}
func (s *Service) PendingInvites(c context.Context, uid int64) ([]business.Invite, error) {
	return s.repo.PendingForUser(c, uid)
}
func (s *Service) OutgoingInvites(c context.Context, uid, bid int64) ([]business.Invite, error) {
	if _, e := s.managed(c, uid, bid); e != nil {
		return nil, e
	}
	return s.repo.PendingForBusiness(c, bid)
}
func (s *Service) respond(c context.Context, uid, iid int64, status string) error {
	i, e := s.repo.Invite(c, iid)
	if e != nil || i.UserID != uid || i.Status != "pending" {
		return business.ErrInviteNotFound
	}
	if status == "accepted" {
		if _, e = s.repo.MemberRole(c, i.BusinessID, uid); e != nil {
			if e = s.repo.AddMember(c, i.BusinessID, uid, i.Role); e != nil {
				return e
			}
		}
	}
	return s.repo.SetInviteStatus(c, iid, status)
}
func (s *Service) AcceptInvite(c context.Context, uid, iid int64) error {
	return s.respond(c, uid, iid, "accepted")
}
func (s *Service) RejectInvite(c context.Context, uid, iid int64) error {
	return s.respond(c, uid, iid, "rejected")
}

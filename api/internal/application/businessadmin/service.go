package businessadmin

import (
	"context"
	"hesab/api/internal/domain/business"
	"hesab/api/internal/domain/user"
	"strings"
	"time"
)

type Owner struct {
	ID                               int64
	FirstName, LastName, PhoneNumber string
}
type ListRow struct {
	business.Business
	Owner       Owner
	MemberCount int64
}
type OwnedRow struct {
	ID          int64
	Name        string
	MemberCount int64
	CreatedAt   time.Time
}
type JoinedRow struct {
	ID                    int64
	Name, Role, OwnerName string
	CreatedAt             time.Time
}
type Repository interface {
	List(context.Context, string, int32, int32) ([]ListRow, error)
	Count(context.Context, string) (int64, error)
	Get(context.Context, int64) (business.Business, Owner, error)
	Members(context.Context, int64) ([]business.Member, error)
	GetUser(context.Context, int64) (user.User, error)
	Create(context.Context, string, int64) (business.Business, error)
	AddMember(context.Context, int64, int64, string) error
	Rename(context.Context, int64, string) (business.Business, error)
	SoftDelete(context.Context, int64) error
	MemberRole(context.Context, int64, int64) (string, error)
	UpdateRole(context.Context, int64, int64, string) error
	RemoveMember(context.Context, int64, int64) error
	ActiveUserByPhone(context.Context, string) (user.User, error)
	Owned(context.Context, int64) ([]OwnedRow, error)
	Joined(context.Context, int64) ([]JoinedRow, error)
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{r} }
func clean(n string) (string, error) {
	n = strings.TrimSpace(n)
	if n == "" {
		return "", business.ErrNameRequired
	}
	return n, nil
}
func (s *Service) List(c context.Context, n string, page, size int) ([]ListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	v, e := s.repo.List(c, strings.TrimSpace(n), int32(size), int32((page-1)*size))
	if e != nil {
		return nil, 0, e
	}
	t, e := s.repo.Count(c, strings.TrimSpace(n))
	return v, t, e
}
func (s *Service) Create(c context.Context, uid int64, n string) (business.Business, error) {
	if _, e := s.repo.GetUser(c, uid); e != nil {
		return business.Business{}, e
	}
	n, e := clean(n)
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
func (s *Service) Get(c context.Context, bid int64) (business.Business, Owner, []business.Member, error) {
	b, o, e := s.repo.Get(c, bid)
	if e != nil {
		return b, o, nil, e
	}
	m, e := s.repo.Members(c, bid)
	return b, o, m, e
}
func (s *Service) Rename(c context.Context, bid int64, n string) (business.Business, error) {
	n, e := clean(n)
	if e != nil {
		return business.Business{}, e
	}
	return s.repo.Rename(c, bid, n)
}
func (s *Service) Delete(c context.Context, bid int64) error { return s.repo.SoftDelete(c, bid) }
func (s *Service) AddMemberByPhone(c context.Context, bid int64, p, role string) (business.Member, error) {
	if !business.AssignableRole(role) {
		return business.Member{}, business.ErrInvalidRole
	}
	p, e := user.NormalizePhone(p)
	if e != nil {
		return business.Member{}, e
	}
	u, e := s.repo.ActiveUserByPhone(c, p)
	if e != nil {
		return business.Member{}, business.ErrInviteeNotRegistered
	}
	if _, e = s.repo.MemberRole(c, bid, u.ID); e == nil {
		return business.Member{}, business.ErrAlreadyMember
	}
	if e = s.repo.AddMember(c, bid, u.ID, role); e != nil {
		return business.Member{}, e
	}
	return business.Member{UserID: u.ID, FirstName: u.FirstName, LastName: u.LastName, PhoneNumber: u.PhoneNumber, Role: role}, nil
}
func (s *Service) ChangeRole(c context.Context, bid, target int64, role string) error {
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
func (s *Service) RemoveMember(c context.Context, bid, target int64) error {
	r, e := s.repo.MemberRole(c, bid, target)
	if e != nil {
		return e
	}
	if r == business.RoleOwner {
		return business.ErrCannotTargetOwner
	}
	return s.repo.RemoveMember(c, bid, target)
}
func (s *Service) UserBusinesses(c context.Context, id int64) ([]OwnedRow, []JoinedRow, error) {
	o, e := s.repo.Owned(c, id)
	if e != nil {
		return nil, nil, e
	}
	j, e := s.repo.Joined(c, id)
	return o, j, e
}

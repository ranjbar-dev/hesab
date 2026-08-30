package usersadmin

import (
	"context"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/domain/user"
)

type NewUser struct{ FirstName, LastName, Email, PhoneNumber, NationalID, AccountType, Password, PasswordHash string }
type Profile struct{ FirstName, LastName, Email, NationalID, AccountType string }
type ListFilter struct {
	FirstName, LastName, Phone, Status string
	CreatedFrom, CreatedTo             *time.Time
	Limit, Offset                      int32
}
type Repository interface {
	Create(context.Context, NewUser) (user.User, error)
	Get(context.Context, int64) (user.User, error)
	List(context.Context, ListFilter) ([]user.User, error)
	Count(context.Context, ListFilter) (int64, error)
	UpdateProfile(context.Context, int64, Profile) (user.User, error)
	SetStatus(context.Context, int64, string) (user.User, error)
	SetPassword(context.Context, int64, string) error
	SoftDelete(context.Context, int64) error
	RevokeAllSessions(context.Context, int64) error
}
type SMS interface {
	Send(context.Context, string, string) error
}
type Service struct {
	repo Repository
	sms  SMS
}

func NewService(repo Repository, sms SMS) *Service { return &Service{repo, sms} }

func cleanProfile(p Profile) (Profile, error) {
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.LastName = strings.TrimSpace(p.LastName)
	p.Email = strings.TrimSpace(p.Email)
	p.NationalID = strings.TrimSpace(p.NationalID)
	p.AccountType = strings.TrimSpace(p.AccountType)
	if p.FirstName == "" || p.LastName == "" {
		return p, user.ErrNameRequired
	}
	if e := user.ValidateNationalID(p.NationalID); e != nil {
		return p, e
	}
	if p.AccountType == "" {
		p.AccountType = user.AccountIndividual
	}
	if !user.ValidAccountType(p.AccountType) {
		return p, user.ErrInvalidAccountType
	}
	return p, nil
}
func (s *Service) Create(c context.Context, in NewUser) (user.User, error) {
	p, e := cleanProfile(Profile{FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, NationalID: in.NationalID, AccountType: in.AccountType})
	if e != nil {
		return user.User{}, e
	}
	phone, e := user.NormalizePhone(in.PhoneNumber)
	if e != nil {
		return user.User{}, e
	}
	if e = user.ValidatePassword(in.Password); e != nil {
		return user.User{}, e
	}
	h, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		return user.User{}, e
	}
	v, e := s.repo.Create(c, NewUser{FirstName: p.FirstName, LastName: p.LastName, Email: p.Email, PhoneNumber: phone, NationalID: p.NationalID, AccountType: p.AccountType, PasswordHash: string(h)})
	if e != nil {
		return user.User{}, e
	}
	if e = s.sms.Send(c, phone, "حساب کاربری شما در سامانه حساب ساخته شد. برای ورود از شماره موبایل خود استفاده کنید."); e != nil {
		log.Printf("send welcome SMS: %v", e)
	} // TODO(sms): real template + provider once sms.ir lands.
	return v, nil
}
func (s *Service) List(c context.Context, f ListFilter, page, pageSize int) ([]user.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	f.Limit = int32(pageSize)
	f.Offset = int32((page - 1) * pageSize)
	v, e := s.repo.List(c, f)
	if e != nil {
		return nil, 0, e
	}
	total, e := s.repo.Count(c, f)
	return v, total, e
}
func (s *Service) Get(c context.Context, id int64) (user.User, error) { return s.repo.Get(c, id) }
func (s *Service) UpdateProfile(c context.Context, id int64, in Profile) (user.User, error) {
	p, e := cleanProfile(in)
	if e != nil {
		return user.User{}, e
	}
	return s.repo.UpdateProfile(c, id, p)
}
func (s *Service) SetStatus(c context.Context, id int64, status string) (user.User, error) {
	if !user.ValidStatus(status) {
		return user.User{}, user.ErrInvalidStatus
	}
	v, e := s.repo.SetStatus(c, id, status)
	if e != nil {
		return user.User{}, e
	}
	if status == user.StatusDisabled {
		if e = s.repo.RevokeAllSessions(c, id); e != nil {
			return user.User{}, e
		}
	}
	return v, nil
}
func (s *Service) ResetPassword(c context.Context, id int64, password string) error {
	if e := user.ValidatePassword(password); e != nil {
		return e
	}
	if _, e := s.repo.Get(c, id); e != nil {
		return e
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	if e = s.repo.SetPassword(c, id, string(h)); e != nil {
		return e
	}
	return s.repo.RevokeAllSessions(c, id)
}
func (s *Service) Delete(c context.Context, id int64) error {
	if _, e := s.repo.Get(c, id); e != nil {
		return e
	}
	if e := s.repo.SoftDelete(c, id); e != nil {
		return e
	}
	return s.repo.RevokeAllSessions(c, id)
}

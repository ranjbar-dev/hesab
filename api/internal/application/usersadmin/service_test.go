package usersadmin

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"hesab/api/internal/domain/user"
)

type fakeRepo struct {
	users            map[int64]user.User
	nextID           int64
	lastFilter       ListFilter
	revoked, deleted int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{users: map[int64]user.User{}, nextID: 1} }
func (r *fakeRepo) Create(_ context.Context, in NewUser) (user.User, error) {
	for _, u := range r.users {
		if u.PhoneNumber == in.PhoneNumber {
			return user.User{}, user.ErrPhoneTaken
		}
	}
	u := user.User{ID: r.nextID, FirstName: in.FirstName, LastName: in.LastName, Email: in.Email, PhoneNumber: in.PhoneNumber, NationalID: in.NationalID, AccountType: in.AccountType, Status: user.StatusActive, PasswordHash: in.PasswordHash}
	r.users[u.ID] = u
	r.nextID++
	return u, nil
}
func (r *fakeRepo) Get(_ context.Context, id int64) (user.User, error) {
	u, ok := r.users[id]
	if !ok || u.DeletedAt != nil {
		return user.User{}, user.ErrUserNotFound
	}
	return u, nil
}
func (r *fakeRepo) List(_ context.Context, f ListFilter) ([]user.User, error) {
	r.lastFilter = f
	var out []user.User
	for _, u := range r.users {
		if u.DeletedAt == nil {
			out = append(out, u)
		}
	}
	return out, nil
}
func (r *fakeRepo) Count(_ context.Context, _ ListFilter) (int64, error) {
	var n int64
	for _, u := range r.users {
		if u.DeletedAt == nil {
			n++
		}
	}
	return n, nil
}
func (r *fakeRepo) UpdateProfile(_ context.Context, id int64, in Profile) (user.User, error) {
	u, e := r.Get(context.Background(), id)
	if e != nil {
		return u, e
	}
	u.FirstName, u.LastName, u.Email, u.NationalID, u.AccountType = in.FirstName, in.LastName, in.Email, in.NationalID, in.AccountType
	r.users[id] = u
	return u, nil
}
func (r *fakeRepo) SetStatus(_ context.Context, id int64, status string) (user.User, error) {
	u, e := r.Get(context.Background(), id)
	if e != nil {
		return u, e
	}
	u.Status = status
	r.users[id] = u
	return u, nil
}
func (r *fakeRepo) SetPassword(_ context.Context, id int64, hash string) error {
	u, e := r.Get(context.Background(), id)
	if e == nil {
		u.PasswordHash = hash
		r.users[id] = u
	}
	return e
}
func (r *fakeRepo) SoftDelete(_ context.Context, id int64) error {
	u, e := r.Get(context.Background(), id)
	if e != nil {
		return e
	}
	now := time.Now()
	u.DeletedAt = &now
	u.Status = user.StatusDisabled
	r.users[id] = u
	r.deleted++
	return nil
}
func (r *fakeRepo) RevokeAllSessions(_ context.Context, _ int64) error { r.revoked++; return nil }

type fakeSMS struct{ calls int }

func (s *fakeSMS) Send(context.Context, string, string) error { s.calls++; return nil }
func validNew(phone string) NewUser {
	return NewUser{FirstName: "نام", LastName: "خانوادگی", PhoneNumber: phone, Password: "StrongPass1", NationalID: "0079059740"}
}

func TestCreate(t *testing.T) {
	r := newFakeRepo()
	sms := &fakeSMS{}
	s := NewService(r, sms)
	u, e := s.Create(context.Background(), validNew("9120000000"))
	if e != nil {
		t.Fatal(e)
	}
	if u.PasswordHash == "StrongPass1" || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("StrongPass1")) != nil {
		t.Fatal("password was not bcrypt hashed")
	}
	if sms.calls != 1 {
		t.Fatal("welcome SMS not sent")
	}
	if _, e = s.Create(context.Background(), validNew("9120000000")); !errors.Is(e, user.ErrPhoneTaken) {
		t.Fatalf("want duplicate error, got %v", e)
	}
	bad := validNew("9120000001")
	bad.Password = "weak"
	if _, e = s.Create(context.Background(), bad); !errors.Is(e, user.ErrWeakPassword) {
		t.Fatalf("want weak password, got %v", e)
	}
	bad = validNew("9120000001")
	bad.NationalID = "123"
	if _, e = s.Create(context.Background(), bad); !errors.Is(e, user.ErrInvalidNationalID) {
		t.Fatalf("want invalid ID, got %v", e)
	}
}
func TestListClampsPage(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r, &fakeSMS{})
	if _, _, e := s.List(context.Background(), ListFilter{}, 3, 10); e != nil {
		t.Fatal(e)
	}
	if r.lastFilter.Limit != 10 || r.lastFilter.Offset != 20 {
		t.Fatalf("want 10/20, got %d/%d", r.lastFilter.Limit, r.lastFilter.Offset)
	}
	if _, _, e := s.List(context.Background(), ListFilter{}, 0, 101); e != nil {
		t.Fatal(e)
	}
	if r.lastFilter.Limit != 100 || r.lastFilter.Offset != 0 {
		t.Fatalf("want 100/0, got %d/%d", r.lastFilter.Limit, r.lastFilter.Offset)
	}
}
func TestStatusResetAndDelete(t *testing.T) {
	r := newFakeRepo()
	s := NewService(r, &fakeSMS{})
	u, e := s.Create(context.Background(), validNew("9120000000"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetStatus(context.Background(), u.ID, user.StatusDisabled); e != nil || r.revoked != 1 {
		t.Fatalf("disable/revoke: %v %d", e, r.revoked)
	}
	if _, e = s.SetStatus(context.Background(), u.ID, "bogus"); !errors.Is(e, user.ErrInvalidStatus) {
		t.Fatalf("want invalid status, got %v", e)
	}
	if e = s.ResetPassword(context.Background(), u.ID, "weak"); !errors.Is(e, user.ErrWeakPassword) {
		t.Fatalf("want weak reset, got %v", e)
	}
	if e = s.ResetPassword(context.Background(), u.ID, "OtherPass1"); e != nil {
		t.Fatal(e)
	}
	if r.revoked != 2 {
		t.Fatal("reset did not revoke")
	}
	stored, _ := r.Get(context.Background(), u.ID)
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("OtherPass1")) != nil {
		t.Fatal("reset hash invalid")
	}
	if e = s.Delete(context.Background(), u.ID); e != nil || r.deleted != 1 || r.revoked != 3 {
		t.Fatalf("delete/revoke: %v %d %d", e, r.deleted, r.revoked)
	}
	if _, e = s.Get(context.Background(), u.ID); !errors.Is(e, user.ErrUserNotFound) {
		t.Fatalf("want deleted not found, got %v", e)
	}
}

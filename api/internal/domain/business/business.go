package business

import (
	"errors"
	"time"
)

const (
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleAccountant = "accountant"
	RoleViewer     = "viewer"
)

var (
	ErrNotFound             = errors.New("business not found")
	ErrNameRequired         = errors.New("business name required")
	ErrInvalidRole          = errors.New("invalid business role")
	ErrNotMember            = errors.New("not business member")
	ErrForbidden            = errors.New("business forbidden")
	ErrAlreadyMember        = errors.New("already member")
	ErrInvitePending        = errors.New("invite pending")
	ErrInviteNotFound       = errors.New("invite not found")
	ErrInviteeNotRegistered = errors.New("invitee not registered")
	ErrCannotTargetOwner    = errors.New("owner immutable")
	ErrOwnerCannotLeave     = errors.New("owner cannot leave")
)

func ValidRole(s string) bool {
	return s == RoleOwner || s == RoleAdmin || s == RoleAccountant || s == RoleViewer
}
func AssignableRole(s string) bool   { return s == RoleAdmin || s == RoleAccountant || s == RoleViewer }
func CanManageMembers(s string) bool { return s == RoleOwner || s == RoleAdmin }

type Business struct {
	ID, OwnerUserID int64
	Name            string
	CreatedAt       time.Time
	DeletedAt       *time.Time
}
type Member struct {
	UserID                                 int64
	Role, FirstName, LastName, PhoneNumber string
	CreatedAt                              time.Time
}
type Invite struct {
	ID, BusinessID                            int64
	BusinessName, Role, Status, InvitedByName string
	UserID                                    int64
	FirstName, LastName, PhoneNumber          string
	CreatedAt                                 time.Time
}

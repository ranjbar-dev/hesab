// ponytail: parallel to adminauth, kept separate on purpose.
package user

import (
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrResetCodeInvalid   = errors.New("reset code invalid")
	ErrRefreshInvalid     = errors.New("refresh invalid")
	ErrTwoFARequired      = errors.New("twofa required")
	ErrTwoFACodeInvalid   = errors.New("twofa code invalid")
	ErrWeakPassword       = errors.New("weak password")
)

type User struct {
	ID                                                                int64
	FirstName, LastName, Email, PhoneNumber, PasswordHash, TOTPSecret string
	NationalID, AccountType, Status                                   string
	DeletedAt                                                         *time.Time
	CreatedAt                                                         time.Time
}

func (u User) TwoFAEnabled() bool { return u.TOTPSecret != "" }

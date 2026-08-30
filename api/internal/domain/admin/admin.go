package admin

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

type Admin struct {
	ID                                                                int64
	FirstName, LastName, Email, PhoneNumber, PasswordHash, TOTPSecret string
	IsMale                                                            bool
	CreatedAt                                                         time.Time
}

func (a Admin) TwoFAEnabled() bool { return a.TOTPSecret != "" }

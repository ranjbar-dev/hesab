package user

import (
	"errors"
	"strconv"
)

const (
	StatusActive      = "active"
	StatusDisabled    = "disabled"
	AccountIndividual = "individual"
	AccountCompany    = "company"
)

var (
	ErrPhoneTaken         = errors.New("phone already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrNameRequired       = errors.New("first and last name required")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidAccountType = errors.New("invalid account type")
	ErrInvalidNationalID  = errors.New("invalid national id")
)

func ValidStatus(s string) bool      { return s == StatusActive || s == StatusDisabled }
func ValidAccountType(s string) bool { return s == AccountIndividual || s == AccountCompany }

func ValidateNationalID(s string) error {
	if s == "" {
		return nil
	}
	if len(s) != 10 {
		return ErrInvalidNationalID
	}
	allSame, sum := true, 0
	for i := 0; i < 9; i++ {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return ErrInvalidNationalID
		}
		if s[i] != s[0] {
			allSame = false
		}
		sum += d * (10 - i)
	}
	check, err := strconv.Atoi(s[9:])
	if err != nil || allSame {
		return ErrInvalidNationalID
	}
	r := sum % 11
	if (r < 2 && check == r) || (r >= 2 && check == 11-r) {
		return nil
	}
	return ErrInvalidNationalID
}

package admin

import "unicode"

func ValidatePassword(s string) error {
	if len(s) < 8 {
		return ErrWeakPassword
	}
	var letter, digit bool
	for _, r := range s {
		letter = letter || unicode.IsLetter(r)
		digit = digit || unicode.IsDigit(r)
	}
	if !letter || !digit {
		return ErrWeakPassword
	}
	return nil
}

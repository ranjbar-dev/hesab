package admin

import (
	"errors"
	"regexp"
	"strings"
)

var phoneRE = regexp.MustCompile(`^9\d{9}$`)

func NormalizePhone(raw string) (string, error) {
	s := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(s, "+98"):
		s = s[3:]
	case strings.HasPrefix(s, "0098"):
		s = s[4:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case strings.HasPrefix(s, "0"):
		s = s[1:]
	}
	if !phoneRE.MatchString(s) {
		return "", errors.New("invalid phone number")
	}
	return s, nil
}

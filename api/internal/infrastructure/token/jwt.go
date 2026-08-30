package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"hesab/api/internal/config"
	"strconv"
	"time"
)

type JWT struct {
	secret                []byte
	accessTTL, pendingTTL time.Duration
}
type claims struct {
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

func New(cfg config.Config) JWT {
	return JWT{[]byte(cfg.JWTSecret), cfg.AccessTokenTTL, cfg.TwoFAPendingTTL}
}
func (j JWT) issue(id int64, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims{typ, jwt.RegisteredClaims{Subject: strconv.FormatInt(id, 10), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl))}}).SignedString(j.secret)
}
func (j JWT) IssueAccess(id int64) (string, int, error) {
	s, e := j.issue(id, "admin", j.accessTTL)
	return s, int(j.accessTTL.Seconds()), e
}
func (j JWT) IssuePending(id int64) (string, error) {
	return j.issue(id, "admin_2fa_pending", j.pendingTTL)
}
func (j JWT) parse(raw, want string) (int64, error) {
	c := new(claims)
	t, e := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("bad signing method")
		}
		return j.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if e != nil || !t.Valid || c.Type != want {
		return 0, errors.New("invalid token")
	}
	return strconv.ParseInt(c.Subject, 10, 64)
}
func (j JWT) ParseAccess(s string) (int64, error)  { return j.parse(s, "admin") }
func (j JWT) ParsePending(s string) (int64, error) { return j.parse(s, "admin_2fa_pending") }
func (j JWT) IssueClientAccess(id int64) (string, int, error) {
	s, e := j.issue(id, "client", j.accessTTL)
	return s, int(j.accessTTL.Seconds()), e
}
func (j JWT) IssueClientPending(id int64) (string, error) {
	return j.issue(id, "client_2fa_pending", j.pendingTTL)
}
func (j JWT) ParseClientAccess(s string) (int64, error)  { return j.parse(s, "client") }
func (j JWT) ParseClientPending(s string) (int64, error) { return j.parse(s, "client_2fa_pending") }

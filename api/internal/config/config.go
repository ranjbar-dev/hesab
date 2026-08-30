package config

import (
	"os"
	"strings"
	"time"
)

// Config holds runtime configuration sourced from the environment.
type Config struct {
	DatabaseURL                                                        string
	Port                                                               string
	JWTSecret                                                          string
	AccessTokenTTL, RefreshTokenTTL, PasswordResetTTL, TwoFAPendingTTL time.Duration
	CORSOrigins                                                        []string
	CookieSecure                                                       bool
	CookieDomain, TOTPIssuer                                           string
}

// Load reads configuration from environment variables, falling back to
// local-development defaults.
func Load() Config {
	return Config{
		DatabaseURL:    getenv("DATABASE_URL", "postgres://hesab:hesab@localhost:5432/hesab?sslmode=disable"),
		Port:           getenv("PORT", "8080"),
		JWTSecret:      getenv("JWT_SECRET", "dev-insecure-admin-secret-change-me"),
		AccessTokenTTL: getdur("ACCESS_TOKEN_TTL", "15m"), RefreshTokenTTL: getdur("REFRESH_TOKEN_TTL", "720h"), PasswordResetTTL: getdur("PASSWORD_RESET_TTL", "5m"), TwoFAPendingTTL: getdur("TWOFA_PENDING_TTL", "5m"),
		CORSOrigins:  strings.Split(getenv("CORS_ORIGINS", "http://localhost:3010,http://localhost:3020"), ","),
		CookieSecure: getenv("COOKIE_SECURE", "true") == "true", CookieDomain: getenv("COOKIE_DOMAIN", ""), TOTPIssuer: getenv("TOTP_ISSUER", "Hesab Admin"),
	}
}

func getdur(key, def string) time.Duration {
	v, err := time.ParseDuration(getenv(key, def))
	if err != nil {
		panic("invalid " + key + ": " + err.Error())
	}
	return v
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

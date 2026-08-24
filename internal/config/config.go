package config

import (
	"errors"
	"os"
	"strconv"
)

// ErrMissingDatabaseURL is returned when DATABASE_URL is unset or blank.
// There is deliberately no fallback: a default would have to spell out a
// username and password in the source tree, and would let the app connect
// somewhere the operator never named rather than saying what is missing.
var ErrMissingDatabaseURL = errors.New("DATABASE_URL is not set (copy .env.example to .env and fill it in)")

type Config struct {
	DatabaseURL       string
	Port              string
	SessionCookieName string
	// SecureCookies gates the Secure attribute on the session cookie. Keep
	// false for local HTTP development; set SECURE_COOKIES=true in
	// production once the app is served over HTTPS, otherwise browsers
	// will silently refuse to store the cookie.
	SecureCookies bool
}

// Load reads the configuration from the environment. Everything but the
// database URL has a safe default, because nothing else here is a secret and
// a missing PORT is not a reason to refuse to start.
func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, ErrMissingDatabaseURL
	}

	return Config{
		DatabaseURL:       databaseURL,
		Port:              getEnv("PORT", "8080"),
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
		SecureCookies:     getEnvBool("SECURE_COOKIES", false),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

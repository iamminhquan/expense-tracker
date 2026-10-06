package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// ErrMissingDatabaseURL is returned when DATABASE_URL is unset or blank.
// There is deliberately no fallback: a default would have to spell out a
// username and password in the source tree, and would let the app connect
// somewhere the operator never named rather than saying what is missing.
var ErrMissingDatabaseURL = errors.New("DATABASE_URL is not set (copy .env.example to .env and fill it in)")

// ErrMissingJWTSecret is returned when JWT_SECRET is unset or blank. Like
// DatabaseURL, this has no fallback: unlike PORT or SESSION_COOKIE_NAME it
// is a secret, and a baked-in default would let every deployment that
// forgets to set it sign tokens with a key an attacker can read straight
// out of the source tree.
var ErrMissingJWTSecret = errors.New("JWT_SECRET is not set (copy .env.example to .env and fill it in)")

// Config holds the application's environment-derived configuration.
type Config struct {
	DatabaseURL       string
	Port              string
	SessionCookieName string
	// SecureCookies gates the Secure attribute on the session cookie. Keep
	// false for local HTTP development; set SECURE_COOKIES=true in
	// production once the app is served over HTTPS, otherwise browsers
	// will silently refuse to store the cookie.
	SecureCookies bool
	// BaseURL is the scheme+host of client/, not server/ -- the
	// password-reset and verification emails link to /reset-password and
	// /verify-email, which are React Router routes the SPA serves, not
	// anything this API answers. Getting this backwards means a perfectly
	// sent email with a link to nowhere (the API has no such GET route).
	BaseURL string
	// BrevoAPIKey and MailFrom configure the Brevo account password-reset
	// email is sent through (see internal/mailer). Both optional: an empty
	// BrevoAPIKey just means mailer.Mailer.Send fails (logged, not fatal)
	// rather than the app refusing to start.
	BrevoAPIKey string
	MailFrom    string
	// JWTSecret signs and verifies access tokens (internal/auth/jwt.go).
	// Required, like DatabaseURL -- see ErrMissingJWTSecret.
	JWTSecret []byte
	// CORSAllowedOrigins lists the origins the Gin API's CORS middleware
	// accepts credentialed cross-origin requests from (the React SPA's
	// dev server and, once deployed, its Vercel domain). Comma-separated
	// in the environment because env vars carry only strings; empty means
	// no cross-origin requests are allowed at all, which is a safe default
	// for a deployment that hasn't set it rather than an error, since a
	// backend-only deployment legitimately has none.
	CORSAllowedOrigins []string
	// TrustedProxies lists the IPs or CIDR ranges of the reverse proxies in
	// front of this server, whose X-Forwarded-For header may be believed
	// when working out a client's address. Empty trusts none, so the client
	// is whoever opened the connection -- right when nothing sits in front,
	// and the safe default otherwise: with it wrong the other way, any
	// client can name itself any address and walk past the per-IP rate
	// limits. Behind a proxy it must be set, or every client shares the
	// proxy's address and one budget.
	TrustedProxies []string
}

// Load reads the configuration from the environment. Everything but the
// database URL and the JWT secret has a safe default, because nothing else
// here is a secret and a missing PORT is not a reason to refuse to start.
func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, ErrMissingDatabaseURL
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, ErrMissingJWTSecret
	}

	port := getEnv("PORT", "8080")

	trustedProxies := getEnvList("TRUSTED_PROXIES")
	for _, p := range trustedProxies {
		if !isIPOrCIDR(p) {
			return Config{}, fmt.Errorf("TRUSTED_PROXIES: %q is not an IP address or CIDR range", p)
		}
	}

	return Config{
		DatabaseURL:       databaseURL,
		Port:              port,
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
		SecureCookies:     getEnvBool("SECURE_COOKIES", false),
		// Default assumes the client/ dev server at Vite's default port,
		// not this server's own port -- see BaseURL's doc comment above.
		BaseURL:            getEnv("APP_BASE_URL", "http://localhost:5173"),
		BrevoAPIKey:        getEnv("BREVO_API_KEY", ""),
		MailFrom:           getEnv("MAIL_FROM", ""),
		JWTSecret:          []byte(jwtSecret),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS"),
		TrustedProxies:     trustedProxies,
	}, nil
}

func isIPOrCIDR(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	_, _, err := net.ParseCIDR(s)
	return err == nil
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

// getEnvList splits a comma-separated environment variable, trimming
// whitespace around each entry and dropping empty ones -- "" and
// "a, ,b" both yield a nil/two-element slice rather than a slice holding
// a stray empty string.
func getEnvList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

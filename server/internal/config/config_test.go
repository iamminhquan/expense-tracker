package config

import (
	"strings"
	"testing"
)

// DATABASE_URL has no default. It used to fall back to a working local
// connection string, which put a username and password in the source tree and
// let the app quietly connect somewhere the operator never named. Refusing to
// start is the honest outcome, and the error has to say which variable is
// missing without echoing anything that looks like a credential.
func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded with DATABASE_URL unset, want an error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not name the missing variable", err)
	}
	if strings.Contains(err.Error(), "postgres://") {
		t.Errorf("error %q suggests a connection string; it must not hand out credentials", err)
	}
}

func TestLoadTakesDatabaseURLFromTheEnvironment(t *testing.T) {
	const dsn = "postgres://someone:somewhere@db.example:5432/app?sslmode=require"
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want no error", err)
	}
	if cfg.DatabaseURL != dsn {
		t.Errorf("DatabaseURL = %q, want it passed through verbatim", cfg.DatabaseURL)
	}
}

// JWT_SECRET has no default, the same reasoning as DatabaseURL above: a
// baked-in signing key would let every deployment that forgets to set it
// mint and verify tokens with a secret sitting in the source tree.
func TestLoadRequiresJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://someone:somewhere@db.example:5432/app")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded with JWT_SECRET unset, want an error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("error %q does not name the missing variable", err)
	}
}

// The remaining fields carry no secret, so they keep their defaults -- a
// missing PORT should not stop the app the way a missing DATABASE_URL or
// JWT_SECRET does.
func TestLoadKeepsTheNonSecretDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://someone:somewhere@db.example:5432/app")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("PORT", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SECURE_COOKIES", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want no error", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.SessionCookieName != "session_id" {
		t.Errorf("SessionCookieName = %q, want session_id", cfg.SessionCookieName)
	}
	if cfg.SecureCookies {
		t.Error("SecureCookies = true, want false by default")
	}
	if cfg.CORSAllowedOrigins != nil {
		t.Errorf("CORSAllowedOrigins = %v, want nil when unset", cfg.CORSAllowedOrigins)
	}
}

func TestLoadSplitsCORSAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://someone:somewhere@db.example:5432/app")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://app.example.com ,,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want no error", err)
	}
	want := []string{"http://localhost:5173", "https://app.example.com"}
	if len(cfg.CORSAllowedOrigins) != len(want) {
		t.Fatalf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, want)
	}
	for i, origin := range want {
		if cfg.CORSAllowedOrigins[i] != origin {
			t.Errorf("CORSAllowedOrigins[%d] = %q, want %q", i, cfg.CORSAllowedOrigins[i], origin)
		}
	}
}

func TestLoadTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "s")

	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 203.0.113.7 ,fc00::/7")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"10.0.0.0/8", "203.0.113.7", "fc00::/7"}
	if len(cfg.TrustedProxies) != len(want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
	for i := range want {
		if cfg.TrustedProxies[i] != want[i] {
			t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
		}
	}

	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,not-an-ip")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted a TRUSTED_PROXIES entry that is neither an IP nor a CIDR")
	}

	t.Setenv("TRUSTED_PROXIES", "")
	cfg, err = Load()
	if err != nil || len(cfg.TrustedProxies) != 0 {
		t.Fatalf("unset TRUSTED_PROXIES = (%v, %v), want none and no error", cfg.TrustedProxies, err)
	}
}

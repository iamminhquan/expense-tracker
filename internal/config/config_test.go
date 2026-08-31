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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want no error", err)
	}
	if cfg.DatabaseURL != dsn {
		t.Errorf("DatabaseURL = %q, want it passed through verbatim", cfg.DatabaseURL)
	}
}

// The other three carry no secret, so they keep their defaults -- a missing
// PORT should not stop the app the way a missing DATABASE_URL does.
func TestLoadKeepsTheNonSecretDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://someone:somewhere@db.example:5432/app")
	t.Setenv("PORT", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("SECURE_COOKIES", "")

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
}

func TestLoadReadsInboundSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("INBOUND_DOMAIN", "in.example.site")
	t.Setenv("INBOUND_WEBHOOK_SECRET", "shh")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.InboundDomain != "in.example.site" {
		t.Errorf("InboundDomain = %q, want %q", cfg.InboundDomain, "in.example.site")
	}
	if cfg.InboundWebhookSecret != "shh" {
		t.Errorf("InboundWebhookSecret = %q, want %q", cfg.InboundWebhookSecret, "shh")
	}
}

// GEMINI_MODEL defaults to a real model name rather than an empty string,
// unlike every other optional setting: internal/classify would otherwise
// have to know the default itself, and this is the one place the app reads
// environment variables at all.
func TestLoadDefaultsGeminiModelWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.GeminiAPIKey != "" {
		t.Errorf("GeminiAPIKey = %q, want empty when unset", cfg.GeminiAPIKey)
	}
	if cfg.GeminiModel != "gemini-3.5-flash-lite" {
		t.Errorf("GeminiModel = %q, want %q", cfg.GeminiModel, "gemini-3.5-flash-lite")
	}
}

func TestLoadReadsGeminiSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-2.5-flash")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.GeminiAPIKey != "test-key" {
		t.Errorf("GeminiAPIKey = %q, want %q", cfg.GeminiAPIKey, "test-key")
	}
	if cfg.GeminiModel != "gemini-2.5-flash" {
		t.Errorf("GeminiModel = %q, want %q", cfg.GeminiModel, "gemini-2.5-flash")
	}
}

func TestLoadLeavesInboundSettingsEmptyWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("INBOUND_DOMAIN", "")
	t.Setenv("INBOUND_WEBHOOK_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.InboundDomain != "" || cfg.InboundWebhookSecret != "" {
		t.Errorf("inbound settings = %q/%q, want both empty", cfg.InboundDomain, cfg.InboundWebhookSecret)
	}
}

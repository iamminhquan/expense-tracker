package handlers_test

import (
	"context"
	"testing"

	"expensetracker/internal/handlers"
)

// themeOf reads the persisted preference straight from the users table, so
// these tests assert on what was actually stored rather than on whatever the
// handler echoed back.
func themeOf(t *testing.T, deps handlers.Deps, email string) string {
	t.Helper()
	var theme string
	err := deps.DB.QueryRow(context.Background(), "SELECT theme FROM users WHERE email = $1", email).Scan(&theme)
	if err != nil {
		t.Fatalf("read theme for %s: %v", email, err)
	}
	return theme
}

func TestRegisterDefaultsThemeToAuto(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "theme-default@example.com"
	loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	if got := themeOf(t, deps, email); got != "auto" {
		t.Fatalf("expected a freshly registered user to default to theme %q, got %q", "auto", got)
	}
}

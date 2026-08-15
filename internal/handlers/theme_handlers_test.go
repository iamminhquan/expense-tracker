package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

// getPage fetches an authenticated page and returns its rendered body.
func getPage(t *testing.T, router http.Handler, cookie *http.Cookie, path string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d", path, rec.Code)
	}
	return rec.Body.String()
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

func TestAuthenticatedPageRendersStoredThemeOnHTMLElement(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "theme-render@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	_, err := deps.DB.Exec(context.Background(), "UPDATE users SET theme = 'dark' WHERE email = $1", email)
	if err != nil {
		t.Fatalf("set theme: %v", err)
	}

	body := getPage(t, router, cookie, "/dashboard")

	if !strings.Contains(body, `<html lang="vi" class="dark">`) {
		t.Fatalf("expected the stored theme to reach the <html> element; got head:\n%s", head(body))
	}
}

// The pre-auth pages carry no user, and html/template renders a missing map
// key as the literal string "<no value>" -- which would land inside the
// class attribute. They must fall back to "auto" instead.
func TestAuthPageRendersAutoThemeWithoutNoValueLeak(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	body := getPage(t, router, nil, "/login")

	if strings.Contains(body, "<no value>") {
		t.Fatalf("login page leaked a missing template key into the output:\n%s", head(body))
	}
	if !strings.Contains(body, `<html lang="vi" class="auto">`) {
		t.Fatalf("expected the login page to fall back to the auto theme; got head:\n%s", head(body))
	}
}

// head trims a rendered page down to its opening markup, so a failure
// message shows the <html> tag instead of dumping the whole document.
func head(body string) string {
	if len(body) > 400 {
		return body[:400]
	}
	return body
}

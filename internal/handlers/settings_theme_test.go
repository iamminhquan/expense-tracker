package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func putTheme(t *testing.T, router http.Handler, cookie *http.Cookie, theme string) *httptest.ResponseRecorder {
	t.Helper()
	tok := csrfTokenFor(t, router)
	form := url.Values{"theme": {theme}}
	req := httptest.NewRequest(http.MethodPut, "/settings/theme", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
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

	if !strings.Contains(body, `<html lang="en" class="dark">`) {
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
	if !strings.Contains(body, `<html lang="en" class="auto">`) {
		t.Fatalf("expected the login page to fall back to the auto theme; got head:\n%s", head(body))
	}
}

func TestUpdateThemeHandlerPersistsPreference(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "theme-persist@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := putTheme(t, router, cookie, "dark")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 storing a valid theme, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := themeOf(t, deps, email); got != "dark" {
		t.Fatalf("expected stored theme %q, got %q", "dark", got)
	}
}

func TestUpdateThemeHandlerRejectsUnknownValue(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "theme-invalid@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	if rec := putTheme(t, router, cookie, "light"); rec.Code != http.StatusNoContent {
		t.Fatalf("setup: expected 204 storing %q, got %d", "light", rec.Code)
	}

	rec := putTheme(t, router, cookie, "neon")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown theme, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := themeOf(t, deps, email); got != "light" {
		t.Fatalf("expected the rejected request to leave theme at %q, got %q", "light", got)
	}
}

func TestUpdateThemeRequiresAuth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	rec := putTheme(t, router, nil, "dark")

	if rec.Code == http.StatusNoContent {
		t.Fatal("expected an unauthenticated theme update to be rejected, got 204")
	}
}

// The switch reflects the stored preference on load, so a user reopening the
// menu sees which mode is actually active rather than a fixed default.
func TestUserMenuMarksTheStoredThemeAsActive(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "theme-switch@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	if rec := putTheme(t, router, cookie, "dark"); rec.Code != http.StatusNoContent {
		t.Fatalf("setup: expected 204, got %d", rec.Code)
	}

	body := getPage(t, router, cookie, "/dashboard")

	sw := themeSwitchMarkup(t, body)
	if !strings.Contains(sw, `data-theme="dark" aria-pressed="true"`) {
		t.Fatalf("expected the dark button to be marked active, got switch:\n%s", sw)
	}
	if strings.Count(sw, `aria-pressed="true"`) != 1 {
		t.Fatalf("expected exactly one active theme button, got switch:\n%s", sw)
	}
}

// themeSwitchMarkup extracts the [data-theme-switch] container so failures
// print the switch rather than the whole page. It anchors on the closing
// angle bracket of the container's own tag, since the bare attribute name
// also appears earlier in the page inside the switch's JS selectors.
func themeSwitchMarkup(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "data-theme-switch>")
	if start < 0 {
		t.Fatalf("no [data-theme-switch] container in the rendered page")
	}
	end := start + 900
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}

// head trims a rendered page down to its opening markup, so a failure
// message shows the <html> tag instead of dumping the whole document.
func head(body string) string {
	if len(body) > 400 {
		return body[:400]
	}
	return body
}

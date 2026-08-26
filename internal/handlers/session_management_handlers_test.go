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

// registerWithUserAgent registers a fresh account the way loginAndGetCookie
// does, but with a caller-chosen User-Agent header, so a test can assert on
// the device label the session list derives from it.
func registerWithUserAgent(t *testing.T, router http.Handler, deps handlers.Deps, email, password, userAgent string) *http.Cookie {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Session Test"}, "email": {email}, "username": {usernameFor(email)}, "password": {password}, "password_confirm": {password}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after register")
	}
	return cookies[0]
}

// TestSettingsPageShowsActiveSessionsAndMarksCurrent covers the reason this
// feature exists: the sessions table already tracked every signed-in device,
// but nothing on /settings ever showed them.
func TestSettingsPageShowsActiveSessionsAndMarksCurrent(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
	cookie := registerWithUserAgent(t, router, deps, "sessions-list@example.com", "s3cret-pass", ua)

	body := settingsBody(t, router, cookie)
	if !strings.Contains(body, "Chrome on Windows") {
		t.Fatalf("expected the session list to show a parsed device label, got: %s", body)
	}
	if !strings.Contains(body, "This device") {
		t.Fatalf("expected the current session to be marked, got: %s", body)
	}
	// The current session must not offer its own revoke button -- there's
	// no reason to let a click log the viewer out of the page they're on.
	if strings.Contains(body, `name="session_id" value="`+cookie.Value+`"`) {
		t.Fatalf("expected no revoke control for the current session, got: %s", body)
	}
}

// TestRevokeOtherSessionsSignsOutOtherDevices covers the "log out everywhere
// else" button, which reuses DeleteOtherSessionsForUser as a deliberate,
// user-triggered action rather than the password-change side effect it was
// written for.
func TestRevokeOtherSessionsSignsOutOtherDevices(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "sessions-revoke-others@example.com"
	cookieA := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a session cookie from the second login")
	}
	cookieB := cookies[0]

	rec2 := postSettings(t, router, cookieA, "/settings/sessions/revoke-others", url.Values{})
	if rec2.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings, got %d: %s", rec2.Code, rec2.Body.String())
	}

	stillIn := func(cookie *http.Cookie) bool {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code == http.StatusOK
	}
	if stillIn(cookieB) {
		t.Fatal("expected the other device's session to be signed out")
	}
	if !stillIn(cookieA) {
		t.Fatal("expected the session that clicked the button to stay logged in")
	}
}

// TestRevokeSingleSessionSignsOutThatDevice covers logging out one listed
// device individually, leaving every other session (including the one
// making the request) untouched.
func TestRevokeSingleSessionSignsOutThatDevice(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "sessions-revoke-one@example.com"
	cookieA := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	cookieB := rec.Result().Cookies()[0]

	rec2 := postSettings(t, router, cookieA, "/settings/sessions/revoke", url.Values{"session_id": {cookieB.Value}})
	if rec2.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings, got %d: %s", rec2.Code, rec2.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req2.AddCookie(cookieB)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req2)
	if rec3.Code == http.StatusOK {
		t.Fatal("expected the targeted session to be signed out")
	}

	req3 := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req3.AddCookie(cookieA)
	rec4 := httptest.NewRecorder()
	router.ServeHTTP(rec4, req3)
	if rec4.Code != http.StatusOK {
		t.Fatal("expected the session making the request to stay logged in")
	}
}

// TestRevokeSessionRejectsAnotherUsersSessionID is the IDOR guard: a
// session_id typed into the form has to be scoped to the caller, or one
// account could sign another one out by guessing or observing an id.
func TestRevokeSessionRejectsAnotherUsersSessionID(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookieVictim := loginAndGetCookie(t, router, deps, "sessions-victim@example.com", "s3cret-pass")
	cookieAttacker := loginAndGetCookie(t, router, deps, "sessions-attacker@example.com", "s3cret-pass")

	postSettings(t, router, cookieAttacker, "/settings/sessions/revoke", url.Values{"session_id": {cookieVictim.Value}})

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(cookieVictim)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal("expected another user's session to survive a foreign revoke attempt")
	}
}

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

// TestSettingsPageRendersTheThreeForms pins the page's shape: profile,
// email and password are three separate forms because each asks for a
// different set of fields -- the email one needs the current password and
// the profile one does not, which a single merged form could only express
// by showing or hiding a field with JavaScript.
func TestSettingsPageRendersTheThreeForms(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "settings-page@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /settings, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, action := range []string{"/settings/profile", "/settings/email", "/settings/password"} {
		if !strings.Contains(body, `action="`+action+`"`) {
			t.Fatalf("expected a form posting to %s, got: %s", action, body)
		}
	}
}

func TestSettingsPageRequiresAuth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected a redirect to /login for an anonymous GET /settings, got %d", rec.Code)
	}
}

// loginAgain logs an existing account in and reports whether the credentials
// were accepted. Success is the HX-Redirect header rather than the status
// code, since loginPage answers 200 either way -- with the dashboard
// redirect on success and the re-rendered form with an error on failure.
func loginAgain(t *testing.T, router http.Handler, email, password string) bool {
	t.Helper()
	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Header().Get("HX-Redirect") != ""
}

// postSettings submits one of the settings forms the way the browser does:
// a plain form POST, which hx-boost turns into an XHR that swaps <body> --
// so the response is a full page render, not a fragment.
func postSettings(t *testing.T, router http.Handler, cookie *http.Cookie, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestChangePasswordAllowsLoginWithNewPasswordOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-change@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/password", url.Values{
		"current_password":     {"s3cret-pass"},
		"new_password":         {"br4nd-new-pass"},
		"new_password_confirm": {"br4nd-new-pass"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings after a successful change, got %d: %s", rec.Code, rec.Body.String())
	}
	if loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the old password to be rejected after the change")
	}
	if !loginAgain(t, router, email, "br4nd-new-pass") {
		t.Fatal("expected the new password to be accepted after the change")
	}
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-wrong-current@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/password", url.Values{
		"current_password":     {"not-my-password"},
		"new_password":         {"br4nd-new-pass"},
		"new_password_confirm": {"br4nd-new-pass"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected the page re-rendered with an error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "current password is not correct") {
		t.Fatalf("expected an error naming the wrong current password, got: %s", rec.Body.String())
	}
	if !loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the original password to still work after a rejected change")
	}
}

func TestChangePasswordRejectsMismatchedConfirmation(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-mismatch@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/password", url.Values{
		"current_password":     {"s3cret-pass"},
		"new_password":         {"br4nd-new-pass"},
		"new_password_confirm": {"br4nd-new-pazz"},
	})

	if !strings.Contains(rec.Body.String(), "do not match") {
		t.Fatalf("expected a mismatch error, got: %s", rec.Body.String())
	}
	if !loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the original password to still work after a rejected change")
	}
}

func TestChangePasswordRejectsShortNewPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-short@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/password", url.Values{
		"current_password":     {"s3cret-pass"},
		"new_password":         {"short"},
		"new_password_confirm": {"short"},
	})

	if !strings.Contains(rec.Body.String(), "at least 8 characters") {
		t.Fatalf("expected a minimum-length error, got: %s", rec.Body.String())
	}
	if !loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the original password to still work after a rejected change")
	}
}

// TestChangePasswordSignsOutOtherSessions covers the reason most people
// change a password in the first place: a session opened somewhere else must
// stop working, while the one doing the changing stays logged in.
func TestChangePasswordSignsOutOtherSessions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-sessions@example.com"
	cookieA := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	// A second login for the same account, standing in for another device.
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

	postSettings(t, router, cookieA, "/settings/password", url.Values{
		"current_password":     {"s3cret-pass"},
		"new_password":         {"br4nd-new-pass"},
		"new_password_confirm": {"br4nd-new-pass"},
	})

	stillIn := func(cookie *http.Cookie) bool {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code == http.StatusOK
	}
	if stillIn(cookieB) {
		t.Fatal("expected the other device's session to be signed out by the password change")
	}
	if !stillIn(cookieA) {
		t.Fatal("expected the session that changed the password to stay logged in")
	}
}

func TestChangePasswordRejectsReusingTheCurrentPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "pw-reuse@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/password", url.Values{
		"current_password":     {"s3cret-pass"},
		"new_password":         {"s3cret-pass"},
		"new_password_confirm": {"s3cret-pass"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected the page re-rendered with an error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "different from the current one") {
		t.Fatalf("expected an error about reusing the current password, got: %s", rec.Body.String())
	}
}

// TestSettingsConfirmsASavedChange covers the other half of the
// redirect-after-post: landing back on /settings has to say something
// happened, or a successful save is indistinguishable from a no-op.
func TestSettingsConfirmsASavedChange(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "settings-saved@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/settings?saved=password", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Password updated") {
		t.Fatalf("expected a confirmation after a saved password change, got: %s", rec.Body.String())
	}
}

// settingsBody fetches the settings page as the logged-in user sees it.
func settingsBody(t *testing.T, router http.Handler, cookie *http.Cookie) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /settings, got %d", rec.Code)
	}
	return rec.Body.String()
}

func TestUpdateProfileChangesNameAndUsername(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "profile-edit@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/profile", url.Values{
		"name":     {"Nguyễn Minh Quân"},
		"username": {"minhquan"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings after saving the profile, got %d: %s", rec.Code, rec.Body.String())
	}
	body := settingsBody(t, router, cookie)
	if !strings.Contains(body, "Nguyễn Minh Quân") {
		t.Fatalf("expected the new name on the settings page, got: %s", body)
	}
	if !strings.Contains(body, `value="minhquan"`) {
		t.Fatalf("expected the new username on the settings page, got: %s", body)
	}
}

// TestUpdateProfileRefreshesTheNavHandle is why these forms are plain POSTs
// rather than htmx fragment swaps: the nav renders the username, and
// hx-boost replaces <body>, so a full-page response keeps the top bar from
// showing a handle the account no longer has.
func TestUpdateProfileRefreshesTheNavHandle(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "profile-nav@example.com", "s3cret-pass")

	postSettings(t, router, cookie, "/settings/profile", url.Values{
		"name":     {"Nav Test"},
		"username": {"nav_handle"},
	})

	body := settingsBody(t, router, cookie)
	if !strings.Contains(body, "nav_handle") {
		t.Fatalf("expected the nav to show the new handle, got: %s", body)
	}
	if strings.Contains(body, "profile_nav") {
		t.Fatalf("expected the old handle to be gone from the page, got: %s", body)
	}
}

func TestUpdateProfileRejectsTakenUsername(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	loginAndGetCookie(t, router, deps, "profile-taken-a@example.com", "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "profile-taken-b@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookieB, "/settings/profile", url.Values{
		"name":     {"Taken"},
		"username": {usernameFor("profile-taken-a@example.com")},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected the page re-rendered with an error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "already taken") {
		t.Fatalf("expected a taken-username error, got: %s", rec.Body.String())
	}
	if !strings.Contains(settingsBody(t, router, cookieB), usernameFor("profile-taken-b@example.com")) {
		t.Fatal("expected the username to be left unchanged after a rejected save")
	}
}

func TestUpdateProfileRejectsInvalidUsername(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "profile-invalid@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/profile", url.Values{
		"name":     {"Invalid"},
		"username": {"Has Spaces"},
	})

	if !strings.Contains(rec.Body.String(), "Username must be") {
		t.Fatalf("expected a username format error, got: %s", rec.Body.String())
	}
}

func TestUpdateProfileRejectsEmptyName(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "profile-noname@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/profile", url.Values{
		"name":     {"   "},
		"username": {"noname_user"},
	})

	if !strings.Contains(rec.Body.String(), "enter your name") {
		t.Fatalf("expected an empty-name error, got: %s", rec.Body.String())
	}
}

// TestChangeEmailHoldsUntilVerifiedThenSwitchesLoginIdentity pins the
// hold-until-verified design: the address a change asks for only becomes
// the login identity once its own verification link has been visited, so a
// mistyped address can never cost the owner the one they can still be
// reached at (see TestChangeEmailMistypedAddressDoesNotAffectLogin).
func TestChangeEmailHoldsUntilVerifiedThenSwitchesLoginIdentity(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	oldEmail := "email-old@example.com"
	newEmail := "email-new@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", newEmail)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", newEmail) })
	cookie := loginAndGetCookie(t, router, deps, oldEmail, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/email", url.Values{
		"email":            {newEmail},
		"current_password": {"s3cret-pass"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings after requesting the change, got %d: %s", rec.Code, rec.Body.String())
	}
	if !loginAgain(t, router, oldEmail, "s3cret-pass") {
		t.Fatal("expected the old email to still log in before the new one is verified")
	}
	if loginAgain(t, router, newEmail, "s3cret-pass") {
		t.Fatal("expected the new email to not log in before it is verified")
	}

	verifyRec := visitVerifyEmail(router, verificationTokenFor(t, deps, newEmail))
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /verify-email, got %d: %s", verifyRec.Code, verifyRec.Body.String())
	}

	if loginAgain(t, router, oldEmail, "s3cret-pass") {
		t.Fatal("expected the old email to stop logging in once the new one is verified")
	}
	if !loginAgain(t, router, newEmail, "s3cret-pass") {
		t.Fatal("expected the verified new email to log in")
	}
}

func TestChangeEmailRequiresTheCurrentPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "email-guard@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/email", url.Values{
		"email":            {"email-hijack@example.com"},
		"current_password": {"not-my-password"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected the page re-rendered with an error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "current password is not correct") {
		t.Fatalf("expected a wrong-password error, got: %s", rec.Body.String())
	}
	if !loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the original email to still log in after a rejected change")
	}
}

func TestChangeEmailRejectsAnAddressAlreadyRegistered(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	taken := "email-taken-a@example.com"
	loginAndGetCookie(t, router, deps, taken, "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "email-taken-b@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookieB, "/settings/email", url.Values{
		"email":            {taken},
		"current_password": {"s3cret-pass"},
	})

	if !strings.Contains(rec.Body.String(), "already registered") {
		t.Fatalf("expected an already-registered error, got: %s", rec.Body.String())
	}
	if !loginAgain(t, router, "email-taken-b@example.com", "s3cret-pass") {
		t.Fatal("expected the original email to still log in after a rejected change")
	}
}

func TestChangeEmailRejectsAnInvalidAddress(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "email-invalid@example.com", "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/email", url.Values{
		"email":            {"not-an-email"},
		"current_password": {"s3cret-pass"},
	})

	if !strings.Contains(rec.Body.String(), "not valid") {
		t.Fatalf("expected an invalid-address error, got: %s", rec.Body.String())
	}
}

// TestUserMenuLinksToSettings covers the entry point: the account menu used
// to carry a "Change password" button wired to nothing at all, which is
// what this page replaces.
func TestUserMenuLinksToSettings(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "menu-link@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `href="/settings"`) {
		t.Fatalf("expected the account menu to link to /settings, got: %s", body)
	}
	if strings.Contains(body, "Change password</button>") {
		t.Fatal("expected the dead Change password button to be gone from the account menu")
	}
}

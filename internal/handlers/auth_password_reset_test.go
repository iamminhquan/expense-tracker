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

// resetTokenFor reads back the token password_reset_tokens holds for email,
// the way the email the app can't send in a test would have carried it.
func resetTokenFor(t *testing.T, deps handlers.Deps, email string) string {
	t.Helper()
	var token string
	err := deps.DB.QueryRow(context.Background(),
		`SELECT prt.token FROM password_reset_tokens prt
		 JOIN users u ON u.id = prt.user_id
		 WHERE u.email = $1`, email).Scan(&token)
	if err != nil {
		t.Fatalf("read reset token for %s: %v", email, err)
	}
	return token
}

func registerUser(t *testing.T, router http.Handler, tok *http.Cookie, email, username, password string) {
	t.Helper()
	form := url.Values{
		"name": {"Reset Test"}, "email": {email}, "username": {username},
		"password": {password}, "password_confirm": {password},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register %s: expected 200, got %d: %s", email, rec.Code, rec.Body.String())
	}
}

func TestForgotPasswordThenResetFlow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "reset-flow@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	registerUser(t, router, tok, email, "reset_flow", "original-pass")

	forgotForm := url.Values{"email": {email}}
	forgotReq := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(forgotForm.Encode()))
	forgotReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(forgotReq, tok)
	forgotRec := httptest.NewRecorder()
	router.ServeHTTP(forgotRec, forgotReq)
	if forgotRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /forgot-password, got %d: %s", forgotRec.Code, forgotRec.Body.String())
	}
	if !strings.Contains(forgotRec.Body.String(), "we've sent a link") {
		t.Fatalf("expected the generic sent message, got: %s", forgotRec.Body.String())
	}

	token := resetTokenFor(t, deps, email)

	getReq := httptest.NewRequest(http.MethodGet, "/reset-password?token="+token, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if strings.Contains(getRec.Body.String(), "invalid or has expired") {
		t.Fatalf("expected a valid token to render the reset form, got: %s", getRec.Body.String())
	}

	resetForm := url.Values{"token": {token}, "password": {"brand-new-pass"}, "password_confirm": {"brand-new-pass"}}
	resetReq := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(resetForm.Encode()))
	resetReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(resetReq, tok)
	resetRec := httptest.NewRecorder()
	router.ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with HX-Redirect after reset, got %d: %s", resetRec.Code, resetRec.Body.String())
	}
	if got := resetRec.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected HX-Redirect: /transactions, got %q", got)
	}
	if len(resetRec.Result().Cookies()) == 0 {
		t.Fatal("expected a fresh session cookie after reset")
	}

	// The old password must no longer work, and the new one must.
	tok2 := csrfTokenFor(t, router)
	oldLogin := url.Values{"email": {email}, "password": {"original-pass"}}
	oldReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(oldLogin.Encode()))
	oldReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(oldReq, tok2)
	oldRec := httptest.NewRecorder()
	router.ServeHTTP(oldRec, oldReq)
	if strings.Contains(oldRec.Header().Get("HX-Redirect"), "/transactions") {
		t.Fatal("expected the pre-reset password to no longer log in")
	}

	newLogin := url.Values{"email": {email}, "password": {"brand-new-pass"}}
	newReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(newLogin.Encode()))
	newReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(newReq, tok2)
	newRec := httptest.NewRecorder()
	router.ServeHTTP(newRec, newReq)
	if got := newRec.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected the reset password to log in, got HX-Redirect %q, body: %s", got, newRec.Body.String())
	}

	// The token is single-use: replaying it must show the invalid state.
	replayReq := httptest.NewRequest(http.MethodGet, "/reset-password?token="+token, nil)
	replayRec := httptest.NewRecorder()
	router.ServeHTTP(replayRec, replayReq)
	if !strings.Contains(replayRec.Body.String(), "invalid or has expired") {
		t.Fatalf("expected a consumed token to show as invalid, got: %s", replayRec.Body.String())
	}
}

// TestForgotPasswordUnknownEmailGivesSameResponse covers the
// enumeration-safety contract: /forgot-password must not reveal whether an
// email is registered, so an unknown address gets the identical response
// text a real one would.
func TestForgotPasswordUnknownEmailGivesSameResponse(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {"never-registered@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "we've sent a link") {
		t.Fatalf("expected the generic sent message for an unknown email too, got: %s", rec.Body.String())
	}
}

func TestResetPasswordWithBogusTokenShowsInvalid(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/reset-password?token=does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "invalid or has expired") {
		t.Fatalf("expected a bogus token to render as invalid, got: %s", rec.Body.String())
	}
}

func TestResetPasswordMismatchShowsError(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "reset-mismatch@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	registerUser(t, router, tok, email, "reset_mismatch", "original-pass")

	forgotForm := url.Values{"email": {email}}
	forgotReq := httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(forgotForm.Encode()))
	forgotReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(forgotReq, tok)
	router.ServeHTTP(httptest.NewRecorder(), forgotReq)

	token := resetTokenFor(t, deps, email)

	resetForm := url.Values{"token": {token}, "password": {"brand-new-pass"}, "password_confirm": {"different-pass"}}
	resetReq := httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(resetForm.Encode()))
	resetReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(resetReq, tok)
	resetRec := httptest.NewRecorder()
	router.ServeHTTP(resetRec, resetReq)

	if !strings.Contains(resetRec.Body.String(), "The two passwords do not match") {
		t.Fatalf("expected a mismatch error, got: %s", resetRec.Body.String())
	}
}

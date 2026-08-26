package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/handlers"
)

const lockoutPassword = "s3cret-pass"

// registerLockoutUser creates a fresh account for a lockout test and removes
// it afterwards, so repeated runs don't collide on the unique email.
func registerLockoutUser(t *testing.T, deps handlers.Deps, router http.Handler, email, username string) {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"name":             {"Lockout"},
		"email":            {email},
		"username":         {username},
		"password":         {lockoutPassword},
		"password_confirm": {lockoutPassword},
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

// postLogin submits the sign-in form once and returns the recorded response.
func postLogin(t *testing.T, router http.Handler, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestLoginLocksAccountAfterFiveWrongPasswords is the throttle's whole point:
// the fifth wrong password stops the account answering at all, and the
// correct password is refused for as long as the lock stands.
func TestLoginLocksAccountAfterFiveWrongPasswords(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "lockout-five@example.com"
	registerLockoutUser(t, deps, router, email, "lockout_five")

	for i := 1; i <= 4; i++ {
		body := postLogin(t, router, email, "wrong-password").Body.String()
		if !strings.Contains(body, "Incorrect email or password") {
			t.Fatalf("attempt %d: expected the incorrect-credentials error, got: %s", i, body)
		}
		if strings.Contains(body, "Too many failed attempts") {
			t.Fatalf("attempt %d: locked out too early, got: %s", i, body)
		}
	}

	fifth := postLogin(t, router, email, "wrong-password").Body.String()
	if !strings.Contains(fifth, "Too many failed attempts") {
		t.Fatalf("fifth wrong password should lock the account, got: %s", fifth)
	}

	locked := postLogin(t, router, email, lockoutPassword)
	if got := locked.Header().Get("HX-Redirect"); got != "" {
		t.Fatalf("the correct password must not sign in while locked, got HX-Redirect: %q", got)
	}
	if !strings.Contains(locked.Body.String(), "Too many failed attempts") {
		t.Fatalf("expected the lock message for the correct password, got: %s", locked.Body.String())
	}
}

// TestLoginWarnsWithRemainingAttempts covers the countdown: the form stays
// quiet for the first mistypes and only starts counting near the limit.
func TestLoginWarnsWithRemainingAttempts(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "lockout-warn@example.com"
	registerLockoutUser(t, deps, router, email, "lockout_warn")

	for i := 1; i <= 3; i++ {
		body := postLogin(t, router, email, "wrong-password").Body.String()
		if i < 3 && strings.Contains(body, "attempts remaining") {
			t.Fatalf("attempt %d: expected no countdown this early, got: %s", i, body)
		}
		if i == 3 && !strings.Contains(body, "2 attempts remaining") {
			t.Fatalf("attempt 3: expected \"2 attempts remaining\", got: %s", body)
		}
	}

	fourth := postLogin(t, router, email, "wrong-password").Body.String()
	if !strings.Contains(fourth, "1 attempt remaining") {
		t.Fatalf("attempt 4: expected \"1 attempt remaining\", got: %s", fourth)
	}
}

// TestLoginSuccessClearsFailedAttempts makes sure the counter is consecutive
// rather than lifetime: signing in correctly wipes what came before, so four
// mistypes spread over a year never add up to a lock.
func TestLoginSuccessClearsFailedAttempts(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "lockout-clear@example.com"
	registerLockoutUser(t, deps, router, email, "lockout_clear")

	for i := 0; i < 4; i++ {
		postLogin(t, router, email, "wrong-password")
	}

	ok := postLogin(t, router, email, lockoutPassword)
	if got := ok.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected a successful sign-in before the limit, got HX-Redirect %q: %s", got, ok.Body.String())
	}

	user, err := deps.Queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.FailedLoginAttempts != 0 {
		t.Errorf("FailedLoginAttempts after a correct password = %d, want 0", user.FailedLoginAttempts)
	}

	for i := 1; i <= 4; i++ {
		body := postLogin(t, router, email, "wrong-password").Body.String()
		if strings.Contains(body, "Too many failed attempts") {
			t.Fatalf("attempt %d after a reset counter should not lock, got: %s", i, body)
		}
	}
}

// TestLoginUnknownEmailIsNotCounted keeps the throttle from leaking more than
// it must: an address with no account gets the plain error, never a countdown
// that would confirm the address exists.
func TestLoginUnknownEmailIsNotCounted(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "lockout-nobody@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	for i := 1; i <= auth.MaxLoginAttempts; i++ {
		body := postLogin(t, router, email, "wrong-password").Body.String()
		if !strings.Contains(body, "Incorrect email or password") {
			t.Fatalf("attempt %d: expected the incorrect-credentials error, got: %s", i, body)
		}
		if strings.Contains(body, "attempts remaining") || strings.Contains(body, "Too many failed attempts") {
			t.Fatalf("attempt %d: an unknown address must not reveal throttle state, got: %s", i, body)
		}
	}
}

// TestLoginLockExpiresOnItsOwn covers the window lapsing: pushing the stamp
// into the past is exactly what the clock does after LockoutWindow, and the
// account must answer again with no intervention.
func TestLoginLockExpiresOnItsOwn(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "lockout-expiry@example.com"
	registerLockoutUser(t, deps, router, email, "lockout_expiry")

	for i := 0; i < 5; i++ {
		postLogin(t, router, email, "wrong-password")
	}

	if _, err := deps.DB.Exec(context.Background(),
		"UPDATE users SET locked_until = now() - interval '1 minute' WHERE email = $1", email); err != nil {
		t.Fatalf("age the lock: %v", err)
	}

	ok := postLogin(t, router, email, lockoutPassword)
	if got := ok.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected sign-in to work once the lock lapsed, got HX-Redirect %q: %s", got, ok.Body.String())
	}
}

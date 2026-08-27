package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// verificationTokenFor reads back the token email_verification_tokens holds
// for the address being proven -- the way the email the app can't send in a
// test would have carried it.
func verificationTokenFor(t *testing.T, deps handlers.Deps, email string) string {
	t.Helper()
	var token string
	err := deps.DB.QueryRow(context.Background(),
		"SELECT token FROM email_verification_tokens WHERE email = $1", email).Scan(&token)
	if err != nil {
		t.Fatalf("read verification token for %s: %v", email, err)
	}
	return token
}

func emailVerifiedFor(t *testing.T, deps handlers.Deps, email string) bool {
	t.Helper()
	var verified bool
	err := deps.DB.QueryRow(context.Background(),
		"SELECT email_verified FROM users WHERE email = $1", email).Scan(&verified)
	if err != nil {
		t.Fatalf("read email_verified for %s: %v", email, err)
	}
	return verified
}

func visitVerifyEmail(router http.Handler, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/verify-email?token="+token, nil))
	return rec
}

func TestRegisterQueuesVerificationTokenAndVisitingLinkMarksVerified(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-signup@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	if emailVerifiedFor(t, deps, email) {
		t.Fatal("expected a freshly registered account to start unverified")
	}

	rec := visitVerifyEmail(router, verificationTokenFor(t, deps, email))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /verify-email, got %d: %s", rec.Code, rec.Body.String())
	}
	if !emailVerifiedFor(t, deps, email) {
		t.Fatal("expected the account to be verified after visiting the link")
	}
}

func TestVerifyEmailWithBogusTokenShowsInvalid(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	rec := visitVerifyEmail(router, "does-not-exist")
	if !strings.Contains(rec.Body.String(), "invalid or has expired") {
		t.Fatalf("expected a bogus token to render as invalid, got: %s", rec.Body.String())
	}
}

func TestVerifyEmailIsSingleUse(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-replay@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	token := verificationTokenFor(t, deps, email)

	visitVerifyEmail(router, token)
	replay := visitVerifyEmail(router, token)
	if !strings.Contains(replay.Body.String(), "invalid or has expired") {
		t.Fatalf("expected a consumed token to show as invalid, got: %s", replay.Body.String())
	}
}

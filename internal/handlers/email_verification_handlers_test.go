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

// TestChangeEmailAllowsResubmittingTheCurrentAddress guards against the
// pre-check in updateEmailHandler mistaking the caller's own row for a
// collision: GetUserByEmail(email) succeeds when email is unchanged too, so
// the check must exclude the caller's own id or a no-op re-save would be
// rejected as "already registered".
func TestChangeEmailAllowsResubmittingTheCurrentAddress(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-noop@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/email", url.Values{
		"email":            {email},
		"current_password": {"s3cret-pass"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "already registered") {
		t.Fatal("expected resubmitting the current address to not be treated as a collision")
	}
}

// TestChangeEmailMistypedAddressDoesNotAffectLogin is the scenario the
// hold-until-verified design exists for: a typo in the new address must
// never cost the owner their ability to log in with the real one.
func TestChangeEmailMistypedAddressDoesNotAffectLogin(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-typo@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/email", url.Values{
		"email":            {"typo-nobody-owns@example.com"},
		"current_password": {"s3cret-pass"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings, got %d: %s", rec.Code, rec.Body.String())
	}

	if !loginAgain(t, router, email, "s3cret-pass") {
		t.Fatal("expected the real email to keep logging in after a mistyped pending change")
	}
}

func TestResendVerificationReissuesTokenAndInvalidatesTheOldOne(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-resend@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	firstToken := verificationTokenFor(t, deps, email)

	rec := postSettings(t, router, cookie, "/resend-verification", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 back to /settings from /resend-verification, got %d: %s", rec.Code, rec.Body.String())
	}

	secondToken := verificationTokenFor(t, deps, email)
	if firstToken == secondToken {
		t.Fatal("expected a resend to issue a fresh token")
	}

	if replay := visitVerifyEmail(router, firstToken); !strings.Contains(replay.Body.String(), "invalid or has expired") {
		t.Fatalf("expected the superseded token to show as invalid, got: %s", replay.Body.String())
	}
	if fresh := visitVerifyEmail(router, secondToken); fresh.Code != http.StatusOK {
		t.Fatalf("expected the reissued token to verify, got %d: %s", fresh.Code, fresh.Body.String())
	}
}

func TestDashboardShowsVerifyEmailBannerUntilVerified(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "verify-banner@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "Verify your email") {
		t.Fatalf("expected the unverified banner on /dashboard, got: %s", rec.Body.String())
	}

	visitVerifyEmail(router, verificationTokenFor(t, deps, email))

	req2 := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if strings.Contains(rec2.Body.String(), "Verify your email") {
		t.Fatalf("expected the banner to disappear once verified, got: %s", rec2.Body.String())
	}
}

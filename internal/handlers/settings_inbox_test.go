package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"
)

// cookieForTestUser logs a registerTestUser account back in over HTTP and
// returns the session cookie. registerTestUser only reports the account's
// id (the inbox webhook tests that added it never need to act as the user
// over HTTP, only to look up its rows), so this is what lets a settings
// test drive /settings/inbox/* as that same account. "s3cret-pass" is the
// fixed password registerTestUser always registers with.
func cookieForTestUser(t *testing.T, router http.Handler, deps handlers.Deps, userID int64) *http.Cookie {
	t.Helper()
	user, err := deps.Queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserByID(%d) error = %v", userID, err)
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{"email": {user.Email}, "password": {"s3cret-pass"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("login as %s: expected a session cookie, got none (status %d, body %q)", user.Email, rec.Code, rec.Body.String())
	}
	return cookies[0]
}

// markFailedBankEmail inserts one bank_emails row already in the 'failed'
// state, standing in for a message the parser couldn't read. It goes
// through Queries.CreateBankEmail rather than a raw INSERT so the row
// always satisfies the same constraints (the unique (user_id, message_id)
// index in particular) that the real webhook path does.
func markFailedBankEmail(t *testing.T, deps handlers.Deps, userID int64) {
	t.Helper()
	_, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID:        userID,
		MessageID:     "<" + t.Name() + "-failed@mail>",
		FromAddress:   "no-reply@mbbank.com.vn",
		Subject:       "Bien dong so du",
		Body:          "could not be parsed",
		Status:        "failed",
		FailureReason: "unrecognized format",
	})
	if err != nil {
		t.Fatalf("CreateBankEmail() error = %v", err)
	}
}

// markPendingBankEmail inserts one bank_emails row in the 'pending' state --
// the state every forwarded email is actually in once F1 sends the real
// From: header, since this slice has no processor to ever move it further.
func markPendingBankEmail(t *testing.T, deps handlers.Deps, userID int64, subject string) {
	t.Helper()
	_, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID:      userID,
		MessageID:   "<" + t.Name() + "-pending@mail>",
		FromAddress: "no-reply@mbbank.com.vn",
		Subject:     subject,
		Body:        "-50,000 VND",
		Status:      "pending",
	})
	if err != nil {
		t.Fatalf("CreateBankEmail() error = %v", err)
	}
}

func TestSettingsHidesTheInboxCardWhenNoDomainIsConfigured(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = ""
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)

	body := settingsBody(t, router, cookie)
	if strings.Contains(body, "Email tracking") {
		t.Error("settings page shows the Email tracking card with no INBOUND_DOMAIN set, want it hidden")
	}
}

func TestEnablingInboxShowsTheForwardingAddress(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)

	postSettings(t, router, cookie, "/settings/inbox/enable", url.Values{})

	user, err := deps.Queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if !user.InboxToken.Valid || user.InboxToken.String == "" {
		t.Fatal("inbox_token after enable = NULL, want a token")
	}
	if body := settingsBody(t, router, cookie); !strings.Contains(body, user.InboxToken.String+"@in.example.site") {
		t.Error("settings page does not show the forwarding address after enabling")
	}
}

func TestEnablingTwiceRotatesTheAddress(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)

	postSettings(t, router, cookie, "/settings/inbox/enable", url.Values{})
	first, _ := deps.Queries.GetUserByID(context.Background(), userID)
	postSettings(t, router, cookie, "/settings/inbox/enable", url.Values{})
	second, _ := deps.Queries.GetUserByID(context.Background(), userID)

	if first.InboxToken.String == second.InboxToken.String {
		t.Errorf("token after regenerate = %q, want a different one", second.InboxToken.String)
	}
}

func TestDisablingInboxClearsTheToken(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)

	postSettings(t, router, cookie, "/settings/inbox/enable", url.Values{})
	postSettings(t, router, cookie, "/settings/inbox/disable", url.Values{})

	user, _ := deps.Queries.GetUserByID(context.Background(), userID)
	if user.InboxToken.Valid {
		t.Errorf("inbox_token after disable = %q, want NULL", user.InboxToken.String)
	}
}

func TestRetryPutsFailedEmailsBackToPending(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)
	enableInbox(t, deps, userID)

	markFailedBankEmail(t, deps, userID)

	postSettings(t, router, cookie, "/settings/inbox/retry", url.Values{})

	rows, err := deps.Queries.ListRecentFailedBankEmails(context.Background(),
		sqlcgen.ListRecentFailedBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentFailedBankEmails() error = %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("failed emails after retry = %d, want 0", len(rows))
	}
}

// TestSettingsCardShowsAPendingEmail is F2: with F1 fixed, a forwarded email
// sits at 'pending' forever in this slice (no processor exists yet), so the
// card has to list it there rather than only ever showing 'failed' rows --
// otherwise the owner forwards an email and the settings page never shows
// they sent anything.
func TestSettingsCardShowsAPendingEmail(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)
	enableInbox(t, deps, userID)

	markPendingBankEmail(t, deps, userID, "Bien dong so du -50,000 VND")

	body := settingsBody(t, router, cookie)
	if !strings.Contains(body, "Bien dong so du -50,000 VND") {
		t.Error("settings page does not show the pending email's subject")
	}
	if !strings.Contains(body, "Recent emails") {
		t.Error("settings page does not show the recent-emails section for a pending email")
	}
}

// TestSettingsCardHidesRetryWhenNothingFailed guards the other half of F2:
// the retry button is scoped to 'failed' rows only, so it must not appear
// just because the card has rows to show.
func TestSettingsCardHidesRetryWhenNothingFailed(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)
	enableInbox(t, deps, userID)

	markPendingBankEmail(t, deps, userID, "Bien dong so du -50,000 VND")

	body := settingsBody(t, router, cookie)
	if strings.Contains(body, "Try these again") {
		t.Error("settings page shows the retry button with no failed emails, want it hidden")
	}
}

// TestSettingsCardShowsRetryWhenSomethingFailed pairs with the test above:
// the button reappears once there is a failed row to retry, alongside a
// pending one that never triggered it.
func TestSettingsCardShowsRetryWhenSomethingFailed(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	cookie := cookieForTestUser(t, router, deps, userID)
	enableInbox(t, deps, userID)

	markPendingBankEmail(t, deps, userID, "Bien dong so du -50,000 VND")
	markFailedBankEmail(t, deps, userID)

	body := settingsBody(t, router, cookie)
	if !strings.Contains(body, "Try these again") {
		t.Error("settings page hides the retry button despite a failed email, want it shown")
	}
}

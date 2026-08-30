package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/csrf"
	"expensetracker/internal/handlers"
	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// postInbox signs body with secret and posts it to /inbox/{token}.
func postInbox(t *testing.T, router http.Handler, secret, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/inbox/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(inbound.SignatureHeader, inbound.Sign(secret, body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func inboxPayload(from, to, subject, messageID, text string) []byte {
	b, _ := json.Marshal(inbound.Payload{
		From: from, To: to, Subject: subject, MessageID: messageID, Text: text,
	})
	return b
}

// enableInbox gives the user an inbox token and returns it.
func enableInbox(t *testing.T, deps handlers.Deps, userID int64) string {
	t.Helper()
	token, err := inbound.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	err = deps.Queries.SetInboxToken(context.Background(), sqlcgen.SetInboxTokenParams{
		ID: userID, InboxToken: pgtype.Text{String: token, Valid: true},
	})
	if err != nil {
		t.Fatalf("SetInboxToken() error = %v", err)
	}
	return token
}

// TestInboxWebhookStoresAForwardedBankEmail also pins the CSRF exemption
// itself: postInbox attaches no csrf_token cookie or X-CSRF-Token header at
// all, so a 200 here is only possible because the exact POST /inbox/{token}
// route is exempt from csrf.Middleware -- see isInboxWebhookRequest.
func TestInboxWebhookStoresAForwardedBankEmail(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mbbank.com.vn", token+"@in.example.site",
		"Bien dong so du", "<m1@mail>", "TK 123 +50,000 VND")
	rec := postInbox(t, router, "s3cret", token, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	n, err := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountBankEmailsForUser() error = %v", err)
	}
	if n != 1 {
		t.Errorf("stored emails = %d, want 1", n)
	}
}

// The original MIME is stored beside the extracted text so that a later fix to
// the extraction can be replayed against what actually arrived. That promise
// failed once already: an HTML-only notice was stored with its entities
// undecoded, and fixing the extractor could not repair the rows already saved.
func TestInboxWebhookStoresTheOriginalMessageBesideTheExtractedText(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	const rawMIME = "Content-Type: text/html\r\n\r\n<p>S&#7889; ti&#7873;n</p>"
	body, err := json.Marshal(inbound.Payload{
		From: "no-reply@mbbank.com.vn", To: token + "@in.example.site",
		Subject: "s", MessageID: "<raw@mail>", Text: "So tien", Raw: rawMIME,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox = %d, want %d", rec.Code, http.StatusOK)
	}

	rows, err := deps.Queries.ListRecentBankEmails(context.Background(),
		sqlcgen.ListRecentBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentBankEmails() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("stored emails = %d, want 1", len(rows))
	}
	if rows[0].RawBody != rawMIME {
		t.Errorf("RawBody = %q, want %q", rows[0].RawBody, rawMIME)
	}
	if rows[0].Body != "So tien" {
		t.Errorf("Body = %q, want %q", rows[0].Body, "So tien")
	}
}

func TestInboxWebhookRejectsABadSignature(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mbbank.com.vn", token+"@in.example.site", "s", "<m2@mail>", "x")
	rec := postInbox(t, router, "the-wrong-secret", token, body)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /inbox with bad signature = %d, want %d", rec.Code, http.StatusForbidden)
	}
	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 0 {
		t.Errorf("stored emails = %d, want 0", n)
	}
}

func TestInboxWebhookRejectsAnUnknownToken(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)

	body := inboxPayload("no-reply@mbbank.com.vn", "nobody@in.example.site", "s", "<m3@mail>", "x")
	rec := postInbox(t, router, "s3cret", "nobody", body)

	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /inbox with unknown token = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestInboxWebhookStoresAStrangerAsIgnoredRatherThanDroppingIt(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("someone@example.com", token+"@in.example.site", "hi", "<m4@mail>", "hello")
	rec := postInbox(t, router, "s3cret", token, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox from a stranger = %d, want %d", rec.Code, http.StatusOK)
	}
	rows, err := deps.Queries.ListRecentFailedBankEmails(context.Background(),
		sqlcgen.ListRecentFailedBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentFailedBankEmails() error = %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("failed emails = %d, want 0 -- a stranger is ignored, not failed", len(rows))
	}
	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 1 {
		t.Errorf("stored emails = %d, want 1 -- an ignored email is still stored", n)
	}
}

// A lookalike domain must not pass the sender check. The match is on the
// "@domain" suffix rather than the bare domain precisely so that a sender at
// notmbbank.com.vn cannot inherit MB's trust by ending with the same letters.
func TestInboxWebhookRejectsALookalikeSenderDomain(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@notmbbank.com.vn", token+"@in.example.site",
		"s", "<look@mail>", "x")
	if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox from a lookalike domain = %d, want %d", rec.Code, http.StatusOK)
	}

	rows, err := deps.Queries.ListRecentBankEmails(context.Background(),
		sqlcgen.ListRecentBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentBankEmails() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("stored emails = %d, want 1", len(rows))
	}
	if rows[0].Status != "ignored" {
		t.Errorf("status for a lookalike sender = %q, want %q", rows[0].Status, "ignored")
	}
}

func TestInboxWebhookStoresTheSameMessageOnlyOnce(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mbbank.com.vn", token+"@in.example.site", "s", "<dup@mail>", "x")
	for i := 0; i < 2; i++ {
		if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
			t.Fatalf("POST #%d = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
	}

	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 1 {
		t.Errorf("stored emails after two identical posts = %d, want 1", n)
	}
}

func TestInboxWebhookRejectsEveryRequestWhenNoSecretIsConfigured(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = ""
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mbbank.com.vn", token+"@in.example.site", "s", "<m5@mail>", "x")
	rec := postInbox(t, router, "", token, body)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /inbox with no configured secret = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestInboxCSRFExemptionRejectsAnExtraPathSegment pins the boundary
// isInboxWebhookRequest draws: the exemption covers exactly POST
// /inbox/{token}, not everything under the /inbox/ prefix. A path with a
// second segment must fall through to the ordinary CSRF check and be
// rejected, the same as it would be for any other mutating route.
func TestInboxCSRFExemptionRejectsAnExtraPathSegment(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)

	body := inboxPayload("no-reply@mbbank.com.vn", "x@in.example.site", "s", "<extra@mail>", "x")
	req := httptest.NewRequest(http.MethodPost, "/inbox/abc/def", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(inbound.SignatureHeader, inbound.Sign("s3cret", body))
	// Deliberately no CSRF cookie/header: if this path were wrongly treated
	// as exempt, it would sail through to a 404 (no such route) instead of
	// being stopped by csrf.Middleware first.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /inbox/abc/def with no CSRF token = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "CSRF") {
		t.Errorf("POST /inbox/abc/def body = %q, want a CSRF rejection message", rec.Body.String())
	}
}

// TestInboxCSRFExemptionDoesNotApplyToGET pins the other half of the
// boundary: only POST is exempt. GET isn't a mutating method, so
// csrf.Middleware never rejects it either way -- what distinguishes an
// exempt request from a guarded one is that the guarded path still runs
// ensureCookie and sets the csrf_token cookie, while csrfExcept's "exempt"
// branch skips csrf.Middleware (and that cookie) entirely.
func TestInboxCSRFExemptionDoesNotApplyToGET(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/inbox/some-token", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			found = true
		}
	}
	if !found {
		t.Errorf("GET /inbox/some-token: expected csrf.Middleware to run and set a %s cookie, got none -- request may be wrongly CSRF-exempt", csrf.CookieName)
	}
}

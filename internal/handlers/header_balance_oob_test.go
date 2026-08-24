package handlers_test

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"

	"github.com/jackc/pgx/v5/pgtype"
)

// createTransactionViaHTMX posts a transaction the way the page's form does
// and returns the fragment htmx gets back.
func createTransactionViaHTMX(t *testing.T, router http.Handler, cookie *http.Cookie, deps handlers.Deps, amount, occurredOn string) string {
	t.Helper()
	cats, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	form := url.Values{
		"category_id": {strconv.FormatInt(firstCategoryOfType(t, cats, "expense").ID, 10)},
		"amount":      {amount},
		"type":        {"expense"},
		"description": {"OOB probe"},
		"occurred_on": {occurredOn},
	}
	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost/transactions")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return html.UnescapeString(rec.Body.String())
}

// today in the user's own timezone, which is the month the header widget
// always reports on regardless of which month the page is browsing.
func todayInVietnam() string {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// The header widget is the only place a balance is shown, so leaving it
// untouched by a mutation means the number a user is looking at goes wrong
// the moment they add anything, and stays wrong until they reload.
//
// Dating the new transaction today keeps the expected figure stable whatever
// month the suite runs in: the fixture's whole history nets to 11,450,000
// either as this month's own rows or as what this month carried in.
func TestCreateTransactionRefreshesTheHeaderBalance(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := carryoverUser(t, router, deps, "oob-header-create@example.com")

	body := createTransactionViaHTMX(t, router, cookie, deps, "450000", todayInVietnam())

	if !strings.Contains(body, "11,000,000₫") {
		t.Errorf("the response carries no refreshed header balance (want 11,000,000₫)")
	}
	if !strings.Contains(body, `hx-swap-oob="true"`) {
		t.Errorf("the header balance is not marked for an out-of-band swap")
	}
}

// Both nav bars render the widget and both are in the DOM at once, so a swap
// that reaches only one of them leaves the other stale at whichever
// breakpoint the user is on.
func TestHeaderBalanceRefreshReachesBothNavBars(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := carryoverUser(t, router, deps, "oob-header-both@example.com")

	body := createTransactionViaHTMX(t, router, cookie, deps, "450000", todayInVietnam())

	for _, id := range []string{"header-balance-desktop", "header-balance-mobile"} {
		if got := strings.Count(body, id); got != 1 {
			t.Errorf("swap target %s appears %d times, want exactly 1", id, got)
		}
	}
	// Each widget prints the figure twice -- once in the summary, once inside
	// the popover -- so reaching both nav bars means at least four, and this
	// asserts the reach rather than the widget's internal markup.
	if got := strings.Count(body, "11,000,000₫"); got < 4 {
		t.Errorf("refreshed balance appears %d times, want at least 4 (two nav bars)", got)
	}
}

// Deleting has to move the number back the other way.
func TestDeleteTransactionRefreshesTheHeaderBalance(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := carryoverUser(t, router, deps, "oob-header-delete@example.com")

	created := createTransactionViaHTMX(t, router, cookie, deps, "450000", todayInVietnam())
	if !strings.Contains(created, "11,000,000₫") {
		t.Fatalf("precondition failed: create did not refresh the header")
	}
	user, err := deps.Queries.GetUserByEmail(context.Background(), "oob-header-delete@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	var id int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT id FROM transactions WHERE user_id = $1 ORDER BY id DESC LIMIT 1", user.ID).Scan(&id); err != nil {
		t.Fatalf("find created transaction: %v", err)
	}

	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodDelete, "/transactions/"+strconv.FormatInt(id, 10), nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://localhost/transactions")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	body := html.UnescapeString(rec.Body.String())
	if !strings.Contains(body, "11,450,000₫") {
		t.Errorf("deleting did not put the header balance back (want 11,450,000₫)")
	}
}

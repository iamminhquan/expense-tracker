package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// seedTransactions inserts n expenses described "Txn 01".."Txn NN", all dated
// the 1st of the current month so every one of them lands inside the month the
// page defaults to.
//
// Sharing a single date is deliberate: the list orders by "occurred_on DESC,
// id DESC", so with the date constant the ordering reduces to
// newest-created-first. Txn NN is therefore the first row of page 1 and Txn 01
// the last row of the final page, which is what lets these tests name an exact
// row and assert which page it belongs to.
func seedTransactions(t *testing.T, deps handlers.Deps, userID, categoryID int64, n int) {
	t.Helper()
	now := time.Now()
	day := pgtype.Date{Time: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), Valid: true}
	for i := 1; i <= n; i++ {
		if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
			UserID: userID, CategoryID: categoryID, Amount: int64(1000 * i), Type: "expense",
			Description: fmt.Sprintf("Txn %02d", i), OccurredOn: day,
		}); err != nil {
			t.Fatalf("seed transaction %d: %v", i, err)
		}
	}
}

// pagingUser registers a user, gives it an expense category to point at, seeds
// n transactions and returns the session cookie to browse them with.
func pagingUser(t *testing.T, deps handlers.Deps, router http.Handler, email string, n int) *http.Cookie {
	t.Helper()
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	user, err := deps.Queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{Int64: user.ID, Valid: true})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	seedTransactions(t, deps, user.ID, firstCategoryOfType(t, categories, "expense").ID, n)
	return cookie
}

// getTransactions fetches the list page (query is appended verbatim, e.g.
// "?page=2") and returns its body.
func getTransactions(t *testing.T, router http.Handler, cookie *http.Cookie, query string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/transactions"+query, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /transactions%s: expected 200, got %d", query, rec.Code)
	}
	return rec.Body.String()
}

func TestTransactionsFirstPageShowsOnlyItsOwnRows(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-first@example.com", 30)

	body := getTransactions(t, router, cookie, "")

	if !strings.Contains(body, "Txn 30") {
		t.Error("expected the newest transaction on page 1")
	}
	if !strings.Contains(body, "Txn 21") {
		t.Error("expected the 10th newest transaction to still be on page 1")
	}
	if strings.Contains(body, "Txn 20") {
		t.Error("expected the 11th newest transaction to be held back for page 2")
	}
}

func TestTransactionsSecondPageShowsTheRemainingRows(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-second@example.com", 30)

	body := getTransactions(t, router, cookie, "?page=2")

	if !strings.Contains(body, "Txn 20") {
		t.Error("expected the 11th newest transaction on page 2")
	}
	if !strings.Contains(body, "Txn 11") {
		t.Error("expected the 20th newest transaction on page 2")
	}
	if strings.Contains(body, "Txn 21") {
		t.Error("expected page 1's rows to be gone from page 2")
	}
	if strings.Contains(body, "Txn 01") {
		t.Error("expected the oldest transaction to be held back for the last page")
	}
}

func TestTransactionsPagerIsAbsentWhenEverythingFitsOnOnePage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-single@example.com", 5)

	body := getTransactions(t, router, cookie, "")

	if strings.Contains(body, "Page 1 of") {
		t.Error("expected no pager when a single page holds everything")
	}
}

func TestTransactionsPagerLabelsTheCurrentPageAndItsNeighbours(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-labels@example.com", 30)

	first := getTransactions(t, router, cookie, "")
	if !strings.Contains(first, "Page 1 of 3") {
		t.Error("expected page 1 to say which page of how many it is")
	}
	if !strings.Contains(first, "page=2") {
		t.Error("expected page 1 to offer a link to page 2")
	}
	if strings.Contains(first, "page=0") {
		t.Error("expected page 1 to offer no previous page")
	}

	last := getTransactions(t, router, cookie, "?page=3")
	if !strings.Contains(last, "Page 3 of 3") {
		t.Error("expected the last page to say which page of how many it is")
	}
	if !strings.Contains(last, "page=2") {
		t.Error("expected the last page to offer a link back to page 2")
	}
	if strings.Contains(last, "page=4") {
		t.Error("expected the last page to offer no next page")
	}
}

func TestTransactionsPagerLinksKeepTheBrowsedMonth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-month@example.com", 30)

	month := time.Now().Format("2006-01")
	body := getTransactions(t, router, cookie, "?month="+month)

	if !strings.Contains(body, "month="+month+"&page=2") {
		t.Errorf("expected the pager link to stay inside %s, got:\n%s", month, body)
	}
}

func TestTransactionsPageParamOutOfRangeFallsBackToARealPage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-clamp@example.com", 30)

	for _, query := range []string{"?page=0", "?page=abc", "?page=-3"} {
		if body := getTransactions(t, router, cookie, query); !strings.Contains(body, "Txn 30") {
			t.Errorf("expected %q to fall back to page 1", query)
		}
	}
	if body := getTransactions(t, router, cookie, "?page=99"); !strings.Contains(body, "Txn 01") {
		t.Error("expected a page past the end to fall back to the last page")
	}
}

func TestTransactionCountReportsTheWholeMonthNotThePage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-count@example.com", 30)

	body := getTransactions(t, router, cookie, "")

	if !strings.Contains(body, "30 transactions") {
		t.Error("expected the count chip to report all 30 of the month's transactions")
	}
}

func TestCreatingFromALaterPageReturnsTheFirstPage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-create@example.com", 30)

	user, err := deps.Queries.GetUserByEmail(context.Background(), "paging-create@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{Int64: user.ID, Valid: true})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}

	form := url.Values{
		"category_id": {strconv.FormatInt(firstCategoryOfType(t, categories, "expense").ID, 10)},
		"amount":      {"77000"},
		"type":        {"expense"},
		"description": {"Added from page two"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.test/transactions?page=2")
	req.AddCookie(cookie)
	withCSRF(req, csrfTokenFor(t, router))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 creating from page 2, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Retarget"); got != "#transactions-month-section" {
		t.Errorf("expected the response to be retargeted at the whole section, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Added from page two") {
		t.Error("expected the new transaction in the response")
	}
	if !strings.Contains(body, "Txn 30") {
		t.Error("expected page 1's rows in the response, not page 2's")
	}
	if strings.Contains(body, "Txn 20") {
		t.Error("expected page 2's rows to be gone from the response")
	}
	// The section swap replaces the list, not the nav, so the balance widget
	// only moves if this response carries its out-of-band update too.
	if !strings.Contains(body, `id="header-balance-desktop" class="contents" hx-swap-oob="true"`) {
		t.Error("expected the response to refresh the nav balance widget")
	}
}

func TestDeletingATransactionRefreshesThePager(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "paging-delete@example.com", 26)

	body := getTransactions(t, router, cookie, "")
	marker := `id="transaction-row-`
	idx := strings.Index(body, marker)
	if idx == -1 {
		t.Fatal("expected a transaction row on the page")
	}
	rest := body[idx+len(marker):]
	txnID := rest[:strings.Index(rest, `"`)]

	req := httptest.NewRequest(http.MethodDelete, "/transactions/"+txnID, nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(cookie)
	withCSRF(req, csrfTokenFor(t, router))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `id="transactions-pager" hx-swap-oob="true"`) {
		t.Errorf("expected the delete response to swap a refreshed pager, got:\n%s", rec.Body.String())
	}
}

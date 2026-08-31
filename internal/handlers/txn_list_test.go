package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestTransactionsPageFiltersByMonthParam(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-filter@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-month-filter@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	pastMonth := time.Now().AddDate(0, -2, 0)
	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 7000, Type: "expense",
		Description: "Old month txn",
		OccurredOn:  pgtype.Date{Time: time.Date(pastMonth.Year(), pastMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	currentReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	currentReq.AddCookie(cookie)
	currentRec := httptest.NewRecorder()
	router.ServeHTTP(currentRec, currentReq)
	if strings.Contains(currentRec.Body.String(), "Old month txn") {
		t.Fatal("expected the current-month view to NOT include a transaction from two months ago")
	}

	monthParam := pastMonth.Format("2006-01")
	pastReq := httptest.NewRequest(http.MethodGet, "/transactions?month="+monthParam, nil)
	pastReq.AddCookie(cookie)
	pastRec := httptest.NewRecorder()
	router.ServeHTTP(pastRec, pastReq)
	if !strings.Contains(pastRec.Body.String(), "Old month txn") {
		t.Fatalf("expected ?month=%s to include the past-month transaction, got: %s", monthParam, pastRec.Body.String())
	}
}

func TestMonthDropdownReturnsFragmentOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-fragment@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?month="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("expected a fragment response with no <html> wrapper, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `id="transactions-month-section"`) {
		t.Fatalf("expected the transactions_month_section fragment, got: %s", rec.Body.String())
	}
}

// The sticky mobile header (title, month picker, + add button) lives inside
// transactions_month_section and keys its content off .ActiveNav. The
// month-dropdown's own hx-get re-renders exactly that fragment, so if the
// handler ever renders it without setting ActiveNav, switching months wipes
// the header from the page until the next full load.
func TestMonthDropdownFragmentKeepsTheMobileHeader(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-header@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?month="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `aria-label="Add transaction"`) {
		t.Errorf("month-switch fragment lost the sticky header's add button: %s", body)
	}
	if !strings.Contains(body, "Transactions") {
		t.Errorf("month-switch fragment lost the sticky header's title: %s", body)
	}
}

// Neither page shows a #balance-card any more: the remaining balance lives
// solely in the header widget next to the user menu, not as a page card.
func TestBalanceAppearsExactlyOncePerPage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-one-balance@example.com", "s3cret-pass")

	for _, path := range []string{"/transactions", "/dashboard"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d", path, rec.Code)
		}
		if got := strings.Count(rec.Body.String(), `id="balance-card"`); got != 0 {
			t.Errorf(`GET %s rendered %d balance cards, want 0`, path, got)
		}
	}
}

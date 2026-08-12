package handlers_test

import (
	"context"
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

func TestDashboardShowsMonthlyTotal(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-test@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	form := url.Values{
		"category_id": {strconv.FormatInt(firstCategoryOfType(t, categories, "expense").ID, 10)},
		"amount":      {"100000"},
		"type":        {"expense"},
		"description": {"Test spend"},
		"occurred_on": {today},
	}
	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Clean up the transaction we just created so it doesn't leak into other
	// tests / the migration teardown, mirroring the pattern used in
	// transaction_handlers_test.go.
	user, err := deps.Queries.GetUserByEmail(context.Background(), "dash-test@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	dashReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	dashReq.AddCookie(cookie)
	dashRec := httptest.NewRecorder()
	router.ServeHTTP(dashRec, dashReq)

	if dashRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", dashRec.Code, dashRec.Body.String())
	}
	if !strings.Contains(dashRec.Body.String(), "100.000") {
		t.Fatal("expected dashboard to reflect the new transaction's amount")
	}
}

func TestDashboardShowsPreviousMonthComparison(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-compare@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "dash-compare@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	now := time.Now()
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 100000, Type: "expense",
		Description: "current", OccurredOn: pgtype.Date{Time: now, Valid: true},
	}); err != nil {
		t.Fatalf("create current-month transaction: %v", err)
	}

	prevMonth := now.AddDate(0, -1, 0)
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 200000, Type: "expense",
		Description: "previous",
		OccurredOn:  pgtype.Date{Time: time.Date(prevMonth.Year(), prevMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatalf("create previous-month transaction: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Tháng trước") {
		t.Fatalf("expected a previous-month comparison line, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "giảm") {
		t.Fatalf("expected 'giảm' (current 100.000 < previous 200.000), got: %s", rec.Body.String())
	}
}

func TestDashboardEmptyStateWhenNoTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-empty@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Chưa đủ dữ liệu để vẽ biểu đồ") {
		t.Fatalf("expected the empty-state message for a brand-new user with no transactions, got: %s", rec.Body.String())
	}
}

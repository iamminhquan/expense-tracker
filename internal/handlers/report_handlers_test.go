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
	if !strings.Contains(dashRec.Body.String(), "100000") {
		t.Fatal("expected dashboard to reflect the new transaction's amount")
	}
}

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

func TestEndToEndRegisterAddTransactionSeeDashboard(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "e2e-test@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}

	form := url.Values{
		"category_id": {strconv.FormatInt(firstCategoryOfType(t, categories, "expense").ID, 10)},
		"amount":      {"25000"},
		"type":        {"expense"},
		"description": {"Trà sữa"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	tok := csrfTokenFor(t, router)
	addReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	addReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addReq.AddCookie(cookie)
	withCSRF(addReq, tok)
	router.ServeHTTP(httptest.NewRecorder(), addReq)

	// Clean up the transaction we just created so it doesn't leak into other
	// tests / the migration teardown, mirroring the pattern used in
	// transaction_handlers_test.go and report_handlers_test.go.
	user, err := deps.Queries.GetUserByEmail(context.Background(), "e2e-test@example.com")
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
		t.Fatalf("expected 200 from dashboard, got %d", dashRec.Code)
	}
	if !strings.Contains(dashRec.Body.String(), "25,000") {
		t.Fatal("expected dashboard total to include the new transaction")
	}
}

package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/api"
)

func TestExportTransactionsCSV(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	categoryID := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	today := time.Now().Format("2006-01-02")

	req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 50000, "type": "expense",
		"occurredOn": today, "description": "Export me",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed POST /api/transactions = %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/transactions/export", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/transactions/export = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q, want text/csv prefix", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q, want an attachment", cd)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Date,Type,Category,Amount,Note") {
		t.Errorf("CSV body missing header row: %q", body)
	}
	if !strings.Contains(body, "Export me") {
		t.Errorf("CSV body missing the seeded transaction's note: %q", body)
	}
	if !strings.Contains(body, "50000") {
		t.Errorf("CSV body missing the bare integer amount: %q", body)
	}
}

func TestExportRequiresAuthentication(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/transactions/export with no token = %d, want 401", rec.Code)
	}
}

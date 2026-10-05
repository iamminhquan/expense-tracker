package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/api"
)

// createTestCategory creates a category of typ for userID via the API
// itself (rather than a direct SQL insert), so these tests also exercise
// that POST /api/categories keeps working against whatever the
// transactions endpoints need from it.
func createTestCategory(t *testing.T, deps api.Deps, router http.Handler, userID int64, name, typ string) int64 {
	t.Helper()
	req := authedRequest(t, deps, http.MethodPost, "/api/categories", userID)
	req.Body = jsonBody(t, map[string]string{"name": name, "type": typ, "color": "#D97757"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create test category: POST /api/categories = %d %s", rec.Code, rec.Body.String())
	}
	created := decodeJSON[struct{ ID int64 }](t, rec)
	return created.ID
}

// Regression test for a real bug a browser smoke test caught: see
// dashboard_handlers_test.go's identical check for the full explanation.
// monthOptions() (availableMonths here) is the one array field this
// endpoint shares the same nil-slice risk on; a brand-new account with no
// transaction history anywhere hits it on its very first page load.
func TestListTransactionsEmptyAccountHasNoNullArrays(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/transactions", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/transactions = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"availableMonths":null`) {
		t.Errorf("availableMonths serialized as JSON null instead of []\nbody: %s", rec.Body.String())
	}
}

func TestCreateAndListTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	categoryID := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	today := time.Now().Format("2006-01-02")

	req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 50000, "type": "expense",
		"occurredOn": today, "description": "Weekly groceries",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/transactions = %d %s, want 201", rec.Code, rec.Body.String())
	}
	created := decodeJSON[struct {
		ID           int64  `json:"id"`
		CategoryName string `json:"categoryName"`
		Amount       int64  `json:"amount"`
	}](t, rec)
	if created.CategoryName != "Groceries" {
		t.Errorf("created transaction categoryName = %q, want %q", created.CategoryName, "Groceries")
	}
	if created.Amount != 50000 {
		t.Errorf("created transaction amount = %d, want 50000", created.Amount)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/transactions", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/transactions = %d %s, want 200", rec.Code, rec.Body.String())
	}
	list := decodeJSON[struct {
		Transactions []struct{ ID int64 } `json:"transactions"`
		TotalCount   int64                `json:"totalCount"`
	}](t, rec)
	if list.TotalCount < 1 {
		t.Errorf("totalCount = %d, want at least 1", list.TotalCount)
	}
	found := false
	for _, txn := range list.Transactions {
		if txn.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("created transaction %d not found in this month's list", created.ID)
	}
}

func TestCreateTransactionRejectsMismatchedCategoryType(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	categoryID := createTestCategory(t, deps, router, userID, "Salary", "income")

	req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 1000, "type": "expense",
		"occurredOn": time.Now().Format("2006-01-02"), "description": "",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/transactions (income category, expense type) = %d %s, want 400", rec.Code, rec.Body.String())
	}
}

func TestUpdateAndDeleteTransaction(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	categoryID := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	today := time.Now().Format("2006-01-02")

	req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 50000, "type": "expense",
		"occurredOn": today, "description": "original",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	created := decodeJSON[struct{ ID int64 }](t, rec)

	req = authedRequest(t, deps, http.MethodPatch, fmt.Sprintf("/api/transactions/%d", created.ID), userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 75000, "occurredOn": today, "description": "updated",
	})
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/transactions/%d = %d %s, want 200", created.ID, rec.Code, rec.Body.String())
	}
	updated := decodeJSON[struct {
		Amount      int64  `json:"amount"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}](t, rec)
	if updated.Amount != 75000 {
		t.Errorf("Amount = %d, want 75000", updated.Amount)
	}
	if updated.Description != "updated" {
		t.Errorf("Description = %q, want %q", updated.Description, "updated")
	}
	if updated.Type != "expense" {
		t.Errorf("Type = %q, want it unchanged at %q", updated.Type, "expense")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodDelete, fmt.Sprintf("/api/transactions/%d", created.ID), userID))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/transactions/%d = %d, want 204", created.ID, rec.Code)
	}

	req = authedRequest(t, deps, http.MethodPatch, fmt.Sprintf("/api/transactions/%d", created.ID), userID)
	req.Body = jsonBody(t, map[string]any{"categoryId": categoryID, "amount": 1, "occurredOn": today})
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PATCH deleted transaction = %d, want 404", rec.Code)
	}
}

func TestCreateTransactionRejectsFutureDate(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	categoryID := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	farFuture := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
	req.Body = jsonBody(t, map[string]any{
		"categoryId": categoryID, "amount": 1000, "type": "expense",
		"occurredOn": farFuture, "description": "",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/transactions (30 days future) = %d %s, want 400", rec.Code, rec.Body.String())
	}
}

func TestListTransactionsFiltersByType(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	expenseCat := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	incomeCat := createTestCategory(t, deps, router, userID, "Salary", "income")
	today := time.Now().Format("2006-01-02")

	for _, tc := range []struct {
		categoryID int64
		typ        string
	}{
		{expenseCat, "expense"}, {incomeCat, "income"},
	} {
		req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
		req.Body = jsonBody(t, map[string]any{
			"categoryId": tc.categoryID, "amount": 1000, "type": tc.typ,
			"occurredOn": today, "description": "",
		})
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed POST /api/transactions (%s) = %d %s", tc.typ, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/transactions?type=income", userID))
	list := decodeJSON[struct {
		Transactions []struct{ Type string } `json:"transactions"`
	}](t, rec)
	for _, txn := range list.Transactions {
		if txn.Type != "income" {
			t.Errorf("?type=income returned a %q transaction", txn.Type)
		}
	}
	if len(list.Transactions) == 0 {
		t.Error("?type=income returned no transactions, want at least the one just created")
	}
}

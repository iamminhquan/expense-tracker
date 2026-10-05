package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/api"
	"expensetracker/internal/auth"
)

// jsonBody JSON-encodes v for attaching to a request built by authedRequest
// (which, unlike doJSON, takes no body of its own since the caller needs
// to set headers on the request before adding one).
func jsonBody(t *testing.T, v any) io.ReadCloser {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	return io.NopCloser(&buf)
}

// authedRequest builds a request carrying a valid access token for userID,
// so category/transaction/etc. tests don't each have to register a user
// through the HTTP layer just to get one.
func authedRequest(t *testing.T, deps api.Deps, method, path string, userID int64) *http.Request {
	t.Helper()
	token, _, err := auth.IssueAccessToken(userID, deps.JWTSecret)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// createTestUser inserts a user row directly (skipping the HTTP register
// flow, which these tests don't need) and returns its ID.
func createTestUser(t *testing.T, deps api.Deps) int64 {
	t.Helper()
	name, email, username, password := testUser(t, deps)
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	var id int64
	err = deps.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password_hash, name, username) VALUES ($1, $2, $3, $4) RETURNING id",
		email, hash, name, username,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	return id
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestCreateAndListCategories(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	req := authedRequest(t, deps, http.MethodPost, "/api/categories", userID)
	req.Body = jsonBody(t, map[string]string{"name": "Groceries", "type": "expense", "color": "#D97757"})
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/categories = %d %s, want 201", rec.Code, rec.Body.String())
	}
	created := decodeJSON[struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		IsDefault bool   `json:"isDefault"`
	}](t, rec)
	if created.Name != "Groceries" {
		t.Errorf("created category name = %q, want %q", created.Name, "Groceries")
	}
	if created.IsDefault {
		t.Error("a user-created category reported isDefault = true")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/categories", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/categories = %d %s, want 200", rec.Code, rec.Body.String())
	}
	list := decodeJSON[struct {
		ExpenseCategories   []struct{ ID int64 } `json:"expenseCategories"`
		HasCustomCategories bool                 `json:"hasCustomCategories"`
	}](t, rec)
	if !list.HasCustomCategories {
		t.Error("hasCustomCategories = false after creating a category")
	}
	found := false
	for _, c := range list.ExpenseCategories {
		if c.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("created category %d not found in expenseCategories %v", created.ID, list.ExpenseCategories)
	}
}

func TestUpdateCategoryRenameAndRecolor(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	req := authedRequest(t, deps, http.MethodPost, "/api/categories", userID)
	req.Body = jsonBody(t, map[string]string{"name": "Old Name", "type": "expense", "color": "#D97757"})
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	created := decodeJSON[struct{ ID int64 }](t, rec)

	rec = httptest.NewRecorder()
	req = authedRequest(t, deps, http.MethodPatch, fmt.Sprintf("/api/categories/%d", created.ID), userID)
	req.Body = jsonBody(t, map[string]string{"name": "New Name", "color": "#5B8DEF"})
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/categories/%d = %d %s, want 200", created.ID, rec.Code, rec.Body.String())
	}
	updated := decodeJSON[struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}](t, rec)
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Color != "#5B8DEF" {
		t.Errorf("Color = %q, want %q", updated.Color, "#5B8DEF")
	}
}

func TestDeleteCategory(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	req := authedRequest(t, deps, http.MethodPost, "/api/categories", userID)
	req.Body = jsonBody(t, map[string]string{"name": "To Delete", "type": "expense", "color": "#D97757"})
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	created := decodeJSON[struct{ ID int64 }](t, rec)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodDelete, fmt.Sprintf("/api/categories/%d", created.ID), userID))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/categories/%d = %d %s, want 204", created.ID, rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = authedRequest(t, deps, http.MethodPatch, fmt.Sprintf("/api/categories/%d", created.ID), userID)
	req.Body = jsonBody(t, map[string]string{"name": "Should Fail"})
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PATCH deleted category = %d, want 404", rec.Code)
	}
}

func TestDeleteDefaultCategoryForbidden(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/categories", userID))
	list := decodeJSON[struct {
		ExpenseCategories []struct {
			ID        int64 `json:"id"`
			IsDefault bool  `json:"isDefault"`
		} `json:"expenseCategories"`
	}](t, rec)

	var defaultID int64
	for _, c := range list.ExpenseCategories {
		if c.IsDefault {
			defaultID = c.ID
			break
		}
	}
	if defaultID == 0 {
		t.Fatal("no default expense category found to test against")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodDelete, fmt.Sprintf("/api/categories/%d", defaultID), userID))
	if rec.Code != http.StatusForbidden {
		t.Errorf("DELETE default category = %d, want 403", rec.Code)
	}
}

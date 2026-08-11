package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	mgr := auth.NewManager(q)

	handler := auth.RequireAuth(mgr, "session_id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without a valid session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestRequireAuthAcceptsValidSession(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	mgr := auth.NewManager(q)
	ctx := context.Background()
	token, _, err := mgr.CreateSession(ctx, userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	called := false
	handler := auth.RequireAuth(mgr, "session_id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID, ok := auth.UserIDFromContext(r.Context())
		if !ok || gotID != userID {
			t.Fatalf("expected user id %d in context, got %d (ok=%v)", userID, gotID, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

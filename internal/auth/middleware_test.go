package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestRequireAuthHtmxRequestGetsHXRedirect covers Finding 5 from the final
// review: a plain 303 to a request carrying HX-Request: true would have
// htmx's XHR transparently follow it, fetch the full /login document, and
// swap it into whatever element the original request was targeting (e.g. a
// single transaction row) instead of navigating the browser -- garbling the
// page instead of sending the user to log back in. An expired/missing
// session on an htmx request must instead get a 200 with an HX-Redirect
// header, which htmx turns into a real top-level navigation.
func TestRequireAuthHtmxRequestGetsHXRedirect(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	mgr := auth.NewManager(q)

	handler := auth.RequireAuth(mgr, "session_id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without a valid session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/transactions/1/edit", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for an htmx request with no valid session, got %d", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "/login" {
		t.Fatalf("expected HX-Redirect: /login, got %q", got)
	}
	if strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("expected an empty/small body, not a full login page, got: %s", rec.Body.String())
	}
}

func TestRequireAuthAcceptsValidSession(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	mgr := auth.NewManager(q)
	ctx := context.Background()
	token, _, err := mgr.CreateSession(ctx, userID, "test-agent/1.0")
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

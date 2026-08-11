package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/handlers"
)

func TestHealthz(t *testing.T) {
	router := handlers.NewRouter(handlers.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/api"
)

// TestHealthzAtRoot pins the one path render.yaml's healthCheckPath and the
// keep-alive cron both probe. It needs no database, so it runs everywhere.
func TestHealthzAtRoot(t *testing.T) {
	router := api.NewRouter(api.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want %d", rec.Code, http.StatusOK)
	}
}

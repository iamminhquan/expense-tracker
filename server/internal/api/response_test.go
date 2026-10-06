package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/api"

	"github.com/gin-gonic/gin"
)

// decodeData asserts rec is a success envelope and returns its data.
func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var env api.JSONResponse[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
	if !env.Success {
		t.Fatalf("response %q: success = false, want true", rec.Body.String())
	}
	return env.Data
}

// decodeError asserts rec is an error envelope with a literal null data, and
// returns its message.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env api.JSONResponse[json.RawMessage]
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
	if env.Success {
		t.Fatalf("response %q: success = true, want false", rec.Body.String())
	}
	if string(env.Data) != "null" {
		t.Fatalf("response %q: data = %s, want null", rec.Body.String(), env.Data)
	}
	if env.Message == "" {
		t.Fatalf("response %q carries no message", rec.Body.String())
	}
	return env.Message
}

func TestEnvelopeOnResponsesGinProducesItself(t *testing.T) {
	router := api.NewRouter(api.Deps{})
	router.GET("/api/v1/boom", func(*gin.Context) { panic("secret panic detail") })

	tests := []struct {
		name        string
		method      string
		path        string
		auth        string
		wantStatus  int
		wantMessage string
	}{
		{"unknown route", http.MethodGet, "/api/v1/nope", "", http.StatusNotFound, "not found"},
		{"wrong method", http.MethodPut, "/api/v1/login", "", http.StatusMethodNotAllowed, "method not allowed"},
		{"panic", http.MethodGet, "/api/v1/boom", "", http.StatusInternalServerError, "internal server error"},
		{"no bearer token", http.MethodGet, "/api/v1/me", "", http.StatusUnauthorized, "missing or malformed Authorization header"},
		{"bad bearer token", http.MethodGet, "/api/v1/me", "Bearer garbage", http.StatusUnauthorized, "invalid or expired access token"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s = %d, want %d; body %q", tc.method, tc.path, rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := decodeError(t, rec); got != tc.wantMessage {
				t.Errorf("%s %s message = %q, want %q", tc.method, tc.path, got, tc.wantMessage)
			}
			if strings.Contains(rec.Body.String(), "secret panic detail") {
				t.Errorf("%s %s leaked the panic value: %q", tc.method, tc.path, rec.Body.String())
			}
		})
	}
}

// TestCORSPreflightSkipsMethodNotAllowed pins that the CORS middleware still
// answers a preflight before the router's 405 handling can see it.
func TestCORSPreflightSkipsMethodNotAllowed(t *testing.T) {
	const origin = "http://localhost:5173"
	router := api.NewRouter(api.Deps{CORSAllowedOrigins: []string{origin}})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/categories", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS /api/v1/categories = %d, want %d; body %q", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, origin)
	}
}

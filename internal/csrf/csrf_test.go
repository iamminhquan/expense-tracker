package csrf_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/csrf"
)

func TestMiddlewareIssuesCookieOnFirstRequest(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrf.CookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a csrf_token cookie to be set on first request")
	}
}

func TestMiddlewareRejectsMutationWithoutToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a POST with no CSRF cookie/header, got %d", rec.Code)
	}
}

func TestMiddlewareAcceptsMatchingHeaderToken(t *testing.T) {
	var innerCalled bool
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("expected a token from the first request")
	}

	postReq := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postReq.Header.Set(csrf.HeaderName, token)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK || !innerCalled {
		t.Fatalf("expected matching header token to be accepted, got %d", postRec.Code)
	}
}

func TestMiddlewareAcceptsMatchingFormToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}

	postReq := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader("csrf_token="+token))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected a matching csrf_token form field to be accepted for a plain form POST, got %d", postRec.Code)
	}
}

func TestMiddlewareRejectsMismatchedToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}

	postReq := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postReq.Header.Set(csrf.HeaderName, "not-the-real-token")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a mismatched token, got %d", postRec.Code)
	}
}

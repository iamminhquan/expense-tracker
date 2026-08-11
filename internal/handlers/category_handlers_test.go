package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

func loginAndGetCookie(t *testing.T, router http.Handler, deps handlers.Deps, email, password string) *http.Cookie {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	form := url.Values{"name": {"Cat Test"}, "email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after register")
	}
	return cookies[0]
}

func TestCreateAndListCategories(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-test@example.com", "s3cret-pass")

	form := url.Values{"name": {"Du lịch"}, "type": {"expense"}, "color": {"#111111"}}
	req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusSeeOther {
		t.Fatalf("expected success status creating category, got %d: %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/categories", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing categories, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), "Du lịch") {
		t.Fatal("expected created category to appear in list page")
	}
	if !strings.Contains(listRec.Body.String(), "Ăn uống") {
		t.Fatal("expected default category to appear in list page")
	}
}

package handlers_test

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/database"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"
)

func newTestDeps(t *testing.T) handlers.Deps {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	tmpl := template.Must(template.ParseGlob("../web/templates/*.html"))

	return handlers.Deps{
		DB:         pool,
		Queries:    sqlcgen.New(pool),
		Sessions:   auth.NewManager(sqlcgen.New(pool)),
		Templates:  tmpl,
		CookieName: "session_id",
	}
}

func TestRegisterThenLoginFlow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	email := "flow-test@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	form := url.Values{
		"name":     {"Flow Test"},
		"email":    {email},
		"password": {"s3cret-pass"},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after register, got %d: %s", rec.Code, rec.Body.String())
	}

	loginForm := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after login, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a session cookie to be set")
	}
}

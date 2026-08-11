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

	templates := map[string]*template.Template{
		"register":     template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/register.html")),
		"login":        template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/login.html")),
		"categories":   template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/categories.html")),
		"transactions": template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/transactions.html")),
	}

	return handlers.Deps{
		DB:         pool,
		Queries:    sqlcgen.New(pool),
		Sessions:   auth.NewManager(sqlcgen.New(pool)),
		Templates:  templates,
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

func TestLoginAndRegisterPagesRenderDistinctContent(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	// Both pages link to each other ("Đăng nhập"/"Đăng ký" both appear as
	// footer link text on both pages), so headings alone aren't a reliable
	// signal. Instead assert on the form's action attribute and on the
	// "name" input field, which only ever appears on the register form.
	loginBody := loginRec.Body.String()
	if !strings.Contains(loginBody, `action="/login"`) {
		t.Fatalf("expected GET /login body to contain a form posting to /login, got: %s", loginBody)
	}
	if strings.Contains(loginBody, `name="name"`) {
		t.Fatalf("expected GET /login body to NOT contain the register-only name field, got: %s", loginBody)
	}

	registerReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)

	registerBody := registerRec.Body.String()
	if !strings.Contains(registerBody, `action="/register"`) {
		t.Fatalf("expected GET /register body to contain a form posting to /register, got: %s", registerBody)
	}
	if !strings.Contains(registerBody, `name="name"`) {
		t.Fatalf("expected GET /register body to contain the name field, got: %s", registerBody)
	}
}

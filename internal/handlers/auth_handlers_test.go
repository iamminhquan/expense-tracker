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
	"expensetracker/internal/csrf"
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
		"auth":         template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/auth.html", "../web/templates/auth_card_body.html")),
		"categories":   template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/categories.html", "../web/templates/category_row.html")),
		"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/transactions.html", "../web/templates/transaction_row.html")),
		"dashboard":    template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/dashboard.html")),
	}

	return handlers.Deps{
		DB:            pool,
		Queries:       sqlcgen.New(pool),
		Sessions:      auth.NewManager(sqlcgen.New(pool)),
		Templates:     templates,
		CookieName:    "session_id",
		SecureCookies: false,
	}
}

// csrfTokenFor issues a GET request to obtain a fresh csrf_token cookie.
// Every mutating request built by a test must attach this cookie's value as
// both the cookie itself and the X-CSRF-Token header via withCSRF, or
// csrf.Middleware (wired into every route in Step 11) rejects it with 403.
func csrfTokenFor(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			return c
		}
	}
	t.Fatal("expected a csrf_token cookie to be set on GET /login")
	return nil
}

// withCSRF attaches the cookie and header a mutating request needs to pass
// csrf.Middleware.
func withCSRF(req *http.Request, tok *http.Cookie) {
	req.AddCookie(tok)
	req.Header.Set(csrf.HeaderName, tok.Value)
}

func TestRegisterThenLoginFlow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	email := "flow-test@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"name":             {"Flow Test"},
		"email":            {email},
		"password":         {"s3cret-pass"},
		"password_confirm": {"s3cret-pass"},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with HX-Redirect after register, got %d: %s", rec.Code, rec.Body.String())
	}

	loginForm := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(loginReq, tok)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with HX-Redirect after login, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	if got := loginRec.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected HX-Redirect: /transactions, got %q", got)
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

	loginBody := loginRec.Body.String()
	if !strings.Contains(loginBody, `hx-post="/login"`) {
		t.Fatalf("expected GET /login body to contain a form posting to /login via htmx, got: %s", loginBody)
	}
	if strings.Contains(loginBody, `name="name"`) {
		t.Fatalf("expected GET /login body to NOT contain the register-only name field, got: %s", loginBody)
	}

	registerReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)

	registerBody := registerRec.Body.String()
	if !strings.Contains(registerBody, `hx-post="/register"`) {
		t.Fatalf("expected GET /register body to contain a form posting to /register via htmx, got: %s", registerBody)
	}
	if !strings.Contains(registerBody, `name="name"`) {
		t.Fatalf("expected GET /register body to contain the name field, got: %s", registerBody)
	}
}

// TestAuthTabSwitchReturnsFragmentOnly covers the htmx tab-switch contract
// from SPEC.md section 2: a request carrying HX-Request must get back just
// the auth_card_body fragment, not a full <html> page, so htmx can swap it
// into #auth-card without a flash of the logo/tagline re-rendering.
func TestAuthTabSwitchReturnsFragmentOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<html") {
		t.Fatalf("expected an htmx fragment response with no <html> wrapper, got: %s", body)
	}
	if !strings.Contains(body, `hx-post="/register"`) {
		t.Fatalf("expected the register form fragment, got: %s", body)
	}
}

// TestLoginSuccessSendsHXRedirect covers PROMPT.md's "every mutation
// returns a fragment, never a full reload" rule applied to the one case
// that needs a real navigation: login/register success signal it via the
// HX-Redirect response header instead of an HTTP 3xx, since htmx treats a
// 3xx as just another response to swap in, not a page navigation.
func TestLoginSuccessSendsHXRedirect(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "hx-redirect@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	regForm := url.Values{"name": {"HX"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"s3cret-pass"}}
	regReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(regForm.Encode()))
	regReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(regReq, tok)
	router.ServeHTTP(httptest.NewRecorder(), regReq)

	tok2 := csrfTokenFor(t, router)
	loginForm := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(loginReq, tok2)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with HX-Redirect on success, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	if got := loginRec.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected HX-Redirect: /transactions, got %q", got)
	}
}

// TestRegisterPasswordMismatchShowsError covers the new "Nhập lại mật khẩu"
// field SPEC.md section 2 adds to the register tab.
func TestRegisterPasswordMismatchShowsError(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "mismatch@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Mismatch"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"different"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering the form, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Mật khẩu nhập lại không khớp") {
		t.Fatalf("expected password-mismatch error, got: %s", rec.Body.String())
	}
	if _, err := deps.Queries.GetUserByEmail(context.Background(), email); err == nil {
		t.Fatal("expected no user to be created when passwords don't match")
	}
}

// TestRegisterValidatesInput covers Finding 4 from the final whole-branch
// review: name/email/password went straight to CreateUser with zero
// validation, so an empty password created a working account. Each case
// below must re-render the register form (200, no session cookie) instead
// of creating a user.
func TestRegisterValidatesInput(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	cases := []struct {
		name string
		form url.Values
	}{
		{
			name: "empty name",
			form: url.Values{"name": {"   "}, "email": {"validate-name@example.com"}, "password": {"s3cret-pass"}},
		},
		{
			name: "invalid email",
			form: url.Values{"name": {"Test"}, "email": {"not-an-email"}, "password": {"s3cret-pass"}},
		},
		{
			name: "short password",
			form: url.Values{"name": {"Test"}, "email": {"validate-pw@example.com"}, "password": {"short"}},
		},
		{
			name: "empty password",
			form: url.Values{"name": {"Test"}, "email": {"validate-empty-pw@example.com"}, "password": {""}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email := tc.form.Get("email")
			deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
			t.Cleanup(func() {
				deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
			})

			tok := csrfTokenFor(t, router)
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			withCSRF(req, tok)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected validation failure to re-render the register form with 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("expected no session cookie to be set when registration is rejected by validation")
			}

			user, err := deps.Queries.GetUserByEmail(context.Background(), email)
			if err == nil {
				t.Fatalf("expected no user to be created for invalid input, but found user id %d", user.ID)
			}
		})
	}
}

// TestRegisterDuplicateEmailShowsSpecificMessage covers the other half of
// Finding 4: only a real unique-constraint violation (Postgres code 23505)
// should show "Email đã được sử dụng" -- other CreateUser errors must not
// be misreported as that.
func TestRegisterDuplicateEmailShowsSpecificMessage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "dup-register@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Dup"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"s3cret-pass"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected the first registration to succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	dupReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(dupReq, tok)
	dupRec := httptest.NewRecorder()
	router.ServeHTTP(dupRec, dupReq)

	if dupRec.Code != http.StatusOK {
		t.Fatalf("expected duplicate registration to re-render the form with 200, got %d: %s", dupRec.Code, dupRec.Body.String())
	}
	if !strings.Contains(dupRec.Body.String(), "Email này đã được dùng.") {
		t.Fatalf("expected duplicate-email message, got: %s", dupRec.Body.String())
	}
}

// TestSessionCookieAttributes covers Finding 3: the session cookie (set on
// register/login) and the clearing cookie (set on logout) must both carry
// SameSite=Lax, and Secure must follow Deps.SecureCookies (false in this
// test's config, matching local HTTP dev).
func TestSessionCookieAttributes(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "cookie-attrs@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Cookie"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"s3cret-pass"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == deps.CookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected a session cookie to be set on register")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected session cookie SameSite=Lax, got %v", sessionCookie.SameSite)
	}
	if sessionCookie.Secure {
		t.Fatal("expected session cookie Secure=false when Deps.SecureCookies is false")
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader("csrf_token="+tok.Value))
	logoutReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	logoutReq.AddCookie(sessionCookie)
	logoutReq.AddCookie(tok)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)

	var clearingCookie *http.Cookie
	for _, c := range logoutRec.Result().Cookies() {
		if c.Name == deps.CookieName {
			clearingCookie = c
		}
	}
	if clearingCookie == nil {
		t.Fatal("expected a clearing cookie to be set on logout")
	}
	if clearingCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected logout clearing cookie SameSite=Lax, got %v", clearingCookie.SameSite)
	}
	if clearingCookie.Secure {
		t.Fatal("expected logout clearing cookie Secure=false when Deps.SecureCookies is false")
	}
}

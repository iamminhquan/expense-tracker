package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds shared dependencies for handlers.
//
// Templates is keyed by page name (e.g. "auth", "categories"). Each entry
// is a *template.Template built from layout.html plus that page's own
// template file(s), so that every template set has only a single
// {{define "content"}} block in scope. This avoids a collision that would
// occur if all page templates were parsed together into one shared
// *template.Template: Go's html/template registers "content" as a single
// global name per template set, so the last-parsed page's block would win
// for every page.
type Deps struct {
	DB         *pgxpool.Pool
	Queries    *sqlcgen.Queries
	Sessions   *auth.Manager
	Templates  map[string]*template.Template
	CookieName string
	// SecureCookies gates the Secure attribute on the session and CSRF
	// cookies; see internal/config.Config.SecureCookies for how it's
	// populated.
	SecureCookies bool
}

// renderAuthFragmentOrPage renders just the auth_card_body fragment when
// the request came from htmx (tab switch, or a validation re-render after
// a failed submit -- both submit forms with hx-post/hx-target="#auth-card"
// so a fragment is exactly what's expected back), or the full auth page
// shell on a direct navigation/refresh/bookmark.
func renderAuthFragmentOrPage(w http.ResponseWriter, r *http.Request, deps Deps, data map[string]any) {
	if r.Header.Get("HX-Request") == "true" {
		renderNamed(w, r, deps, "auth", "auth_card_body", "", data)
		return
	}
	render(w, r, deps, "auth", "", data)
}

func loginPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderAuthFragmentOrPage(w, r, deps, map[string]any{"Tab": "login"})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := deps.Queries.GetUserByEmail(r.Context(), email)
		if err != nil || !auth.VerifyPassword(user.PasswordHash, password) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "login", "Error": "Email hoặc mật khẩu không đúng.", "Email": email,
			})
			return
		}

		startSession(w, r, deps, user.ID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

func registerPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderAuthFragmentOrPage(w, r, deps, map[string]any{"Tab": "register"})
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		fail := func(msg string) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "register", "Error": msg, "Name": name, "Email": email,
			})
		}

		if name == "" {
			fail("Vui lòng nhập họ tên")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			fail("Email không hợp lệ")
			return
		}
		if len([]rune(password)) < 8 {
			fail("Mật khẩu phải có ít nhất 8 ký tự.")
			return
		}
		if password != passwordConfirm {
			fail("Mật khẩu nhập lại không khớp.")
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			fail("Không thể tạo tài khoản")
			return
		}

		user, err := deps.Queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				fail("Email này đã được dùng.")
				return
			}
			log.Printf("register: create user: %v", err)
			fail("Không thể tạo tài khoản, vui lòng thử lại")
			return
		}

		startSession(w, r, deps, user.ID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

func logoutHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(deps.CookieName); err == nil {
			deps.Sessions.DeleteSession(r.Context(), cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     deps.CookieName,
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Secure:   deps.SecureCookies,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func startSession(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	token, expiresAt, err := deps.Sessions.CreateSession(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     deps.CookieName,
		Value:    token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   deps.SecureCookies,
	})
}

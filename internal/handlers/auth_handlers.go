package handlers

import (
	"html/template"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds shared dependencies for handlers.
//
// Templates is keyed by page name (e.g. "register", "login"). Each entry is
// a *template.Template built from layout.html plus exactly one page's own
// template file, so that every template set has only a single
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
}

func registerPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			deps.Templates["register"].ExecuteTemplate(w, "layout", map[string]any{})
			return
		}

		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		hash, err := auth.HashPassword(password)
		if err != nil {
			deps.Templates["register"].ExecuteTemplate(w, "layout", map[string]any{"Error": "Không thể tạo tài khoản", "Name": name, "Email": email})
			return
		}

		user, err := deps.Queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
		})
		if err != nil {
			deps.Templates["register"].ExecuteTemplate(w, "layout", map[string]any{"Error": "Email đã được sử dụng", "Name": name, "Email": email})
			return
		}

		startSession(w, r, deps, user.ID)
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}

func loginPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			deps.Templates["login"].ExecuteTemplate(w, "layout", map[string]any{})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := deps.Queries.GetUserByEmail(r.Context(), email)
		if err != nil || !auth.VerifyPassword(user.PasswordHash, password) {
			deps.Templates["login"].ExecuteTemplate(w, "layout", map[string]any{"Error": "Email hoặc mật khẩu không đúng", "Email": email})
			return
		}

		startSession(w, r, deps, user.ID)
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}

func logoutHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(deps.CookieName); err == nil {
			deps.Sessions.DeleteSession(r.Context(), cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{Name: deps.CookieName, Value: "", MaxAge: -1, Path: "/"})
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
	})
}

package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// RequireAuth returns a middleware that validates the session cookie and
// injects the authenticated user's ID into the request context.
func RequireAuth(mgr *Manager, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			userID, err := mgr.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// redirectToLogin sends the browser to /login. For an htmx request, a plain
// 3xx http.Redirect is the wrong tool: htmx's XHR transparently follows it,
// receives the full /login HTML document, and swaps that whole page into
// whatever element the original request was targeting (e.g. a single
// transaction row mid-edit) instead of navigating the browser -- producing
// a garbled page rather than sending the user to log back in. Sessions last
// 7 days, so hitting an expired session mid-interaction is a normal case,
// not an edge case. Mirrors the HX-Redirect pattern already used for the
// login/register success paths in auth_handlers.go: for an HX-Request, set
// the HX-Redirect header instead, which htmx reads and turns into a full
// top-level navigation.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// UserIDFromContext extracts the authenticated user's ID from the request context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDContextKey).(int64)
	return id, ok
}

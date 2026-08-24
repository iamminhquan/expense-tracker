package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
)

// settingsPage renders the account settings page. The three cards it shows
// are three independent forms rather than one: only the email change asks
// for the current password, and a merged form could express that difference
// only by showing or hiding a field with JavaScript.
func settingsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := settingsData(r, deps)
		if err != nil {
			log.Printf("settings: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		render(w, r, deps, "settings", "settings", data)
	}
}

// savedMessages turns the ?saved= marker each successful POST redirects
// with back into a confirmation line. The redirect is what keeps a reload
// or a back button from re-submitting the form, and this is what stops the
// page it lands on from looking like nothing happened.
var savedMessages = map[string]string{
	"profile":  "Profile updated.",
	"email":    "Email updated.",
	"password": "Password updated.",
}

// settingsData loads the current values the three forms are pre-filled with.
func settingsData(r *http.Request, deps Deps) (map[string]any, error) {
	userID, _ := auth.UserIDFromContext(r.Context())
	user, err := deps.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ProfileName":     user.Name,
		"ProfileUsername": user.Username,
		"ProfileEmail":    user.Email,
		"Saved":           savedMessages[r.URL.Query().Get("saved")],
	}, nil
}

// renderSettingsError re-renders the whole settings page with one form's
// error message. Errors answer 200 with the page rather than a fragment:
// hx-boost swaps <body>, so the browser lands on the same page it submitted
// from, with the message in place and the other two forms untouched.
func renderSettingsError(w http.ResponseWriter, r *http.Request, deps Deps, field, msg string) {
	data, err := settingsData(r, deps)
	if err != nil {
		log.Printf("settings: load user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data[field] = msg
	render(w, r, deps, "settings", "settings", data)
}

// updatePasswordHandler changes the account password and signs out every
// other session, which is the reason most people change one at all. The
// current session survives: the person doing the changing should not have
// to log back in a second later.
func updatePasswordHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		current := r.FormValue("current_password")
		next := r.FormValue("new_password")
		confirm := r.FormValue("new_password_confirm")

		user, err := deps.Queries.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("update password: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if !auth.VerifyPassword(user.PasswordHash, current) {
			renderSettingsError(w, r, deps, "PasswordError", "That current password is not correct.")
			return
		}
		if len([]rune(next)) < 8 {
			renderSettingsError(w, r, deps, "PasswordError", "New password must be at least 8 characters.")
			return
		}
		if next != confirm {
			renderSettingsError(w, r, deps, "PasswordError", "The two new passwords do not match.")
			return
		}
		if next == current {
			renderSettingsError(w, r, deps, "PasswordError", "The new password must be different from the current one.")
			return
		}

		hash, err := auth.HashPassword(next)
		if err != nil {
			log.Printf("update password: hash: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.Queries.UpdateUserPassword(r.Context(), sqlcgen.UpdateUserPasswordParams{
			ID: userID, PasswordHash: hash,
		}); err != nil {
			log.Printf("update password: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// The cookie is read straight off the request rather than carried in
		// the context: RequireAuth injects the user id it resolved, not the
		// session id it resolved it from, and this is the only handler that
		// needs to tell one session apart from the rest.
		if cookie, err := r.Cookie(deps.CookieName); err == nil {
			if err := deps.Queries.DeleteOtherSessionsForUser(r.Context(), sqlcgen.DeleteOtherSessionsForUserParams{
				UserID: userID, ID: cookie.Value,
			}); err != nil {
				log.Printf("update password: delete other sessions: %v", err)
			}
		}

		http.Redirect(w, r, "/settings?saved=password", http.StatusSeeOther)
	}
}

// updateProfileHandler saves the two fields that identify the account to
// its owner rather than to the login form: the free-text name and the
// handle the nav shows. Neither asks for the password -- only the email
// change does, since that is what someone logs in with.
func updateProfileHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		name := strings.TrimSpace(r.FormValue("name"))
		username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))

		if name == "" {
			renderSettingsError(w, r, deps, "ProfileError", "Please enter your name.")
			return
		}
		if !usernamePattern.MatchString(username) {
			renderSettingsError(w, r, deps, "ProfileError", "Username must be 3-20 characters: lowercase letters, numbers, or underscores, starting with a letter.")
			return
		}

		err := deps.Queries.UpdateUserProfile(r.Context(), sqlcgen.UpdateUserProfileParams{
			ID: userID, Name: name, Username: username,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				renderSettingsError(w, r, deps, "ProfileError", "That username is already taken.")
				return
			}
			log.Printf("update profile: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/settings?saved=profile", http.StatusSeeOther)
	}
}

// updateEmailHandler changes the address the account logs in with, which is
// why it is the one settings form that asks for the current password: a
// borrowed unlocked browser should not be enough to take the account over.
func updateEmailHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		email := strings.TrimSpace(r.FormValue("email"))
		current := r.FormValue("current_password")

		user, err := deps.Queries.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("update email: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if !auth.VerifyPassword(user.PasswordHash, current) {
			renderSettingsError(w, r, deps, "EmailError", "That current password is not correct.")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			renderSettingsError(w, r, deps, "EmailError", "That email address is not valid.")
			return
		}

		if err := deps.Queries.UpdateUserEmail(r.Context(), sqlcgen.UpdateUserEmailParams{
			ID: userID, Email: email,
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				renderSettingsError(w, r, deps, "EmailError", "That email is already registered.")
				return
			}
			log.Printf("update email: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/settings?saved=email", http.StatusSeeOther)
	}
}

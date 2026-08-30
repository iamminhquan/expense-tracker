package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
	"profile":           "Profile updated.",
	"email-pending":     "Check your inbox to confirm the new address.",
	"verification-sent": "Verification email sent.",
	"password":          "Password updated.",
	"session-revoked":   "Signed out of that session.",
	"sessions-revoked":  "Signed out of every other session.",
	"inbox-enabled":     "Email tracking is on. Forward your bank email to the address below.",
	"inbox-disabled":    "Email tracking is off. The old address no longer accepts mail.",
	"inbox-retried":     "Those emails are set back to pending.",
}

// sessionView is what the settings template shows for one row of the
// active-sessions list: a device label instead of the raw User-Agent, a
// local-time stamp instead of the raw timestamp, and whether this is the
// session the page is being viewed from.
type sessionView struct {
	ID        string
	Device    string
	CreatedAt string
	IsCurrent bool
}

// settingsData loads the current values the three forms are pre-filled with.
func settingsData(r *http.Request, deps Deps) (map[string]any, error) {
	userID, _ := auth.UserIDFromContext(r.Context())
	user, err := deps.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}

	currentToken := ""
	if cookie, err := r.Cookie(deps.CookieName); err == nil {
		currentToken = cookie.Value
	}
	sessions, err := deps.Queries.ListSessionsForUser(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	views := make([]sessionView, 0, len(sessions))
	for _, s := range sessions {
		views = append(views, sessionView{
			ID:        s.ID,
			Device:    format.DeviceLabel(s.UserAgent.String),
			CreatedAt: format.Timestamp(s.CreatedAt, vietnamLocation),
			IsCurrent: s.ID == currentToken,
		})
	}

	data := map[string]any{
		"ProfileName":     user.Name,
		"ProfileUsername": user.Username,
		"ProfileEmail":    user.Email,
		"PendingEmail":    user.PendingEmail.String,
		"Sessions":        views,
		"Saved":           savedMessages[r.URL.Query().Get("saved")],
	}

	inboxData, err := inboxSettingsData(r, deps, userID, user.InboxToken)
	if err != nil {
		return nil, err
	}
	for k, v := range inboxData {
		data[k] = v
	}

	return data, nil
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

// updateEmailHandler stages the address the account would log in with next,
// which is why it is the one settings form that asks for the current
// password: a borrowed unlocked browser should not be enough to take the
// account over. It does not touch users.email itself -- ApplyVerifiedEmail
// (see auth_email_verification.go) does that once the link this queues
// has been visited -- so a mistyped address can never cost the owner the
// one they can still be reached at.
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
		// pending_email carries no UNIQUE constraint of its own -- it is
		// staging data, not the source of truth -- so a collision has to be
		// checked explicitly here rather than caught off a write. The real
		// guarantee is still users.email's constraint, enforced when the
		// link is confirmed; this is just what turns a collision into an
		// immediate, friendly error instead of a dead link discovered later.
		// Excluding the caller's own row is what lets a no-op resubmit of
		// the address already on the account through, the way the old
		// UPDATE-based check always did.
		if existing, err := deps.Queries.GetUserByEmail(r.Context(), email); err == nil && existing.ID != userID {
			renderSettingsError(w, r, deps, "EmailError", "That email is already registered.")
			return
		}

		if err := deps.Queries.SetPendingEmail(r.Context(), sqlcgen.SetPendingEmailParams{
			ID: userID, PendingEmail: pgtype.Text{String: email, Valid: true},
		}); err != nil {
			log.Printf("update email: set pending: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		queueVerificationEmail(r.Context(), deps, userID, email)

		http.Redirect(w, r, "/settings?saved=email-pending", http.StatusSeeOther)
	}
}

// revokeSessionHandler signs one listed device out. DeleteSessionForUser is
// scoped to the caller's own user_id, so a session_id copied or guessed from
// another account can never delete a row it doesn't own.
func revokeSessionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		sessionID := r.FormValue("session_id")

		if err := deps.Queries.DeleteSessionForUser(r.Context(), sqlcgen.DeleteSessionForUserParams{
			ID: sessionID, UserID: userID,
		}); err != nil {
			log.Printf("revoke session: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/settings?saved=session-revoked", http.StatusSeeOther)
	}
}

// revokeOtherSessionsHandler is the "log out everywhere else" button. It
// calls the same query updatePasswordHandler uses as a side effect, but here
// as a deliberate action the person asked for.
func revokeOtherSessionsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		cookie, err := r.Cookie(deps.CookieName)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.Queries.DeleteOtherSessionsForUser(r.Context(), sqlcgen.DeleteOtherSessionsForUserParams{
			UserID: userID, ID: cookie.Value,
		}); err != nil {
			log.Printf("revoke other sessions: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/settings?saved=sessions-revoked", http.StatusSeeOther)
	}
}

// deleteAccountHandler erases the account and everything it owns. It is the
// only action in the app that cannot be undone, so it is gated on the
// current password -- the same gate the email and password changes use --
// rather than on holding a live session alone.
func deleteAccountHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		user, err := deps.Queries.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("delete account: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !auth.VerifyPassword(user.PasswordHash, r.FormValue("current_password")) {
			renderSettingsError(w, r, deps, "DeleteError", "That password is not correct.")
			return
		}

		if err := deleteAccount(r.Context(), deps, userID); err != nil {
			log.Printf("delete account: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Every session row went with the user row; what is left is the
		// browser's own copy of the cookie, which would otherwise be sent
		// on the next request and resolve to nothing.
		clearSessionCookie(w, deps)
		http.Redirect(w, r, "/login?deleted=1", http.StatusSeeOther)
	}
}

// deleteAccount removes an account's rows in dependency order, in one
// transaction so a failure part-way leaves the account whole.
//
// The order is spelled out rather than left to the ON DELETE CASCADE on
// users: transactions.category_id references categories(id) with no ON
// DELETE clause at all, so what makes a single cascading delete work is
// that Postgres defers that NO ACTION check to the end of the statement.
// True, but invisible to anyone reading the delete later, and it stops
// being true the moment someone marks that constraint RESTRICT.
func deleteAccount(ctx context.Context, deps Deps, userID int64) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := deps.Queries.WithTx(tx)

	if err := qtx.DeleteTransactionsForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete transactions: %w", err)
	}
	// bank_emails.user_id already carries ON DELETE CASCADE from users, so
	// this delete is not what stops a foreign-key error the way the ones
	// around it are -- it is spelled out for the same reason the rest of
	// this function is: leaving it to the cascade works today, but that
	// fact is invisible here and one constraint change away from not being
	// true. transactions.bank_email_id is ON DELETE SET NULL, not NO
	// ACTION, so it never blocks this either way -- there is no ordering
	// bug here to "fix" by moving this call again.
	if err := qtx.DeleteBankEmailsForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete bank emails: %w", err)
	}
	if err := qtx.DeletePersonalCategoriesForUser(ctx, pgInt64(userID)); err != nil {
		return fmt.Errorf("delete categories: %w", err)
	}
	if err := qtx.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return tx.Commit(ctx)
}

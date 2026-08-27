package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// usernamePattern mirrors the 000009 migration's CHECK constraint: a
// lowercase letter followed by 2-19 lowercase letters, digits, or
// underscores. Kept in sync by hand since html/template can't read a DB
// constraint, and duplicated here (rather than shared) because a Go regexp
// and a Postgres one are different enough dialects that sharing the string
// wouldn't buy much.
var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,19}$`)

// renderAuthFragmentOrPage renders just the auth_card_body fragment when
// the request came from htmx (tab switch, or a validation re-render after
// a failed submit -- both submit forms with hx-post/hx-target="#auth-card"
// so a fragment is exactly what's expected back), or the full auth page
// shell on a direct navigation/refresh/bookmark.
func renderAuthFragmentOrPage(w http.ResponseWriter, r *http.Request, deps Deps, data map[string]any) {
	if isFragmentRequest(r) {
		renderNamed(w, r, deps, "auth", "auth_card_body", "", data)
		return
	}
	render(w, r, deps, "auth", "", data)
}

// badCredentials is the answer to every sign-in that fails for a reason the
// visitor is not entitled to know the shape of -- a wrong password, an
// address with no account behind it.
const badCredentials = "Incorrect email or password."

func loginPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			data := map[string]any{"Tab": "login"}
			// The account this marker belongs to no longer exists, so this
			// page is the only place its owner can still be told the
			// deletion went through rather than silently logged out.
			if r.URL.Query().Get("deleted") != "" {
				data["Notice"] = "Your account has been deleted."
			}
			renderAuthFragmentOrPage(w, r, deps, data)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		fail := func(msg string) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "login", "Error": msg, "Email": email,
			})
		}

		// An address with no account is refused without being counted: there
		// is no row to count it against, and a countdown here would answer
		// the question of which addresses are registered.
		user, err := deps.Queries.GetUserByEmail(r.Context(), email)
		if err != nil {
			fail(badCredentials)
			return
		}

		// The lock is checked before the password, so guessing at a locked
		// account can neither be told apart from guessing wrong nor push the
		// window further out.
		if left := auth.LockedFor(user.LockedUntil, time.Now()); left > 0 {
			fail(lockedMessage(left))
			return
		}

		if !auth.VerifyPassword(user.PasswordHash, password) {
			fail(recordFailedAttempt(r.Context(), deps, user))
			return
		}

		clearThrottle(r.Context(), deps, user)
		startSession(w, r, deps, user.ID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

// recordFailedAttempt counts one wrong password against user and returns the
// message the sign-in form should show for it.
func recordFailedAttempt(ctx context.Context, deps Deps, user sqlcgen.User) string {
	// A lapsed lock still carries the count that caused it. Wiping it before
	// counting is what gives someone returning after the window a fresh set
	// of attempts instead of re-locking on their first mistype.
	if user.LockedUntil.Valid {
		clearThrottle(ctx, deps, user)
		user.FailedLoginAttempts = 0
	}

	row, err := deps.Queries.RecordFailedLogin(ctx, sqlcgen.RecordFailedLoginParams{
		MaxAttempts: auth.MaxLoginAttempts,
		LockedUntil: pgtype.Timestamptz{Time: time.Now().Add(auth.LockoutWindow), Valid: true},
		ID:          user.ID,
	})
	if err != nil {
		// The throttle is best-effort against a database that won't answer:
		// refuse the sign-in, but don't turn a failed UPDATE into a 500.
		log.Printf("login: record failed attempt: %v", err)
		return badCredentials
	}

	if left := auth.LockedFor(row.LockedUntil, time.Now()); left > 0 {
		return lockedMessage(left)
	}

	remaining := auth.AttemptsRemaining(row.FailedLoginAttempts)
	if remaining > auth.WarnAtRemaining {
		return badCredentials
	}
	if remaining == 1 {
		return badCredentials + " 1 attempt remaining."
	}
	return fmt.Sprintf("%s %d attempts remaining.", badCredentials, remaining)
}

// clearThrottle resets a user's throttle state, skipping the write for the
// overwhelmingly common case of an account that never had any.
func clearThrottle(ctx context.Context, deps Deps, user sqlcgen.User) {
	if user.FailedLoginAttempts == 0 && !user.LockedUntil.Valid {
		return
	}
	if err := deps.Queries.ClearFailedLogins(ctx, user.ID); err != nil {
		log.Printf("login: clear failed attempts: %v", err)
	}
}

// lockedMessage says how long is left on a lock, in whole minutes rounded up
// by auth.LockMinutes so the number is never optimistic.
func lockedMessage(left time.Duration) string {
	if minutes := auth.LockMinutes(left); minutes != 1 {
		return fmt.Sprintf("Too many failed attempts. Try again in %d minutes.", minutes)
	}
	return "Too many failed attempts. Try again in 1 minute."
}

func registerPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderAuthFragmentOrPage(w, r, deps, map[string]any{"Tab": "register"})
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		fail := func(msg string) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "register", "Error": msg, "Name": name, "Email": email, "Username": username,
			})
		}

		if name == "" {
			fail("Please enter your name.")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			fail("That email address is not valid.")
			return
		}
		if !usernamePattern.MatchString(username) {
			fail("Username must be 3-20 characters: lowercase letters, numbers, or underscores, starting with a letter.")
			return
		}
		if len([]rune(password)) < 8 {
			fail("Password must be at least 8 characters.")
			return
		}
		if password != passwordConfirm {
			fail("The two passwords do not match.")
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			fail("Could not create your account.")
			return
		}

		user, err := deps.Queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
			Username:     username,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "users_username_key" {
					fail("That username is already taken.")
					return
				}
				fail("That email is already registered.")
				return
			}
			log.Printf("register: create user: %v", err)
			fail("Could not create your account, please try again.")
			return
		}

		queueVerificationEmail(r.Context(), deps, user.ID, user.Email)
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
		clearSessionCookie(w, deps)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// clearSessionCookie expires the browser's copy of the session cookie. It
// is separate from deleting the session row because the two have different
// callers: logging out deletes one row, deleting an account takes every row
// with the user, and both still have to tell the browser to forget it.
func clearSessionCookie(w http.ResponseWriter, deps Deps) {
	http.SetCookie(w, &http.Cookie{
		Name:     deps.CookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   deps.SecureCookies,
	})
}

func startSession(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	token, expiresAt, err := deps.Sessions.CreateSession(r.Context(), userID, r.UserAgent())
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

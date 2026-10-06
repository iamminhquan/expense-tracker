package api

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

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// usernamePattern mirrors the 000009 migration's CHECK constraint -- see
// handlers/auth_handlers.go's identical pattern for why it's duplicated
// rather than shared (a Go regexp and a Postgres one are different enough
// dialects that sharing the string wouldn't buy much).
var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,19}$`)

// vietnamLocation duplicates handlers/req_month.go's helper of the same
// name -- this package will need it again once the dashboard/transactions
// endpoints land (month-bounded queries), at which point it's worth
// promoting to a shared package rather than duplicating a third time.
var vietnamLocation = loadVietnamLocation()

func loadVietnamLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}

// badCredentials is the answer to every sign-in that fails for a reason the
// caller is not entitled to know the shape of -- a wrong password, an
// address with no account behind it. Matches handlers.badCredentials
// verbatim so a client showing both UIs during the migration can't tell
// the two apart by wording.
const badCredentials = "Incorrect email or password."

// userDTO is the user-facing shape returned by auth endpoints and GET
// /api/v1/me. It deliberately excludes PasswordHash, FailedLoginAttempts, and
// LockedUntil -- internal account-security state no client needs.
type userDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	// Theme rides along here rather than only in GET /api/v1/settings: the
	// client applies it as soon as it knows who's signed in (on
	// bootstrap, via /api/v1/refresh or /api/v1/me), the same moment
	// handlers.authPageView loads it for every authenticated page's nav
	// on the HTML side. Waiting for a separate /api/v1/settings call would
	// mean every page flashes the wrong theme before it's ready.
	Theme string `json:"theme"`
}

func newUserDTO(u sqlcgen.User) userDTO {
	return userDTO{ID: u.ID, Name: u.Name, Email: u.Email, Username: u.Username, Theme: u.Theme}
}

// authResponse is what register/login/refresh all return: a fresh access
// token the client holds in memory (never persisted -- see the migration
// plan's locked JWT-storage decision) plus the user it names. The refresh
// token itself never appears in a JSON body; it only ever travels as the
// httpOnly cookie setRefreshCookie sets.
type authResponse struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	User        userDTO   `json:"user"`
}

type registerRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

func registerHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Email = strings.TrimSpace(req.Email)
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))

		switch {
		case req.Name == "":
			respondError(c, http.StatusBadRequest, "please enter your name")
			return
		case func() bool { _, err := mail.ParseAddress(req.Email); return err != nil }():
			respondError(c, http.StatusBadRequest, "that email address is not valid")
			return
		case !usernamePattern.MatchString(req.Username):
			respondError(c, http.StatusBadRequest, "username must be 3-20 characters: lowercase letters, numbers, or underscores, starting with a letter")
			return
		case len([]rune(req.Password)) < 8:
			respondError(c, http.StatusBadRequest, "password must be at least 8 characters")
			return
		case req.Password != req.PasswordConfirm:
			respondError(c, http.StatusBadRequest, "the two passwords do not match")
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			log.Printf("register: hash password: %v", err)
			respondError(c, http.StatusInternalServerError, "could not create your account")
			return
		}

		user, err := deps.Queries.CreateUser(c.Request.Context(), sqlcgen.CreateUserParams{
			Email:        req.Email,
			PasswordHash: hash,
			Name:         req.Name,
			Username:     req.Username,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "users_username_key" {
					respondError(c, http.StatusConflict, "that username is already taken")
					return
				}
				respondError(c, http.StatusConflict, "that email is already registered")
				return
			}
			log.Printf("register: create user: %v", err)
			respondError(c, http.StatusInternalServerError, "could not create your account, please try again")
			return
		}

		queueVerificationEmail(c.Request.Context(), deps, user.ID, user.Email)
		issueAuthResponse(c, deps, user, "account created")
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}

		// An address with no account is refused without being counted: there
		// is no row to count it against, and a countdown here would answer
		// the question of which addresses are registered.
		user, err := deps.Queries.GetUserByEmail(c.Request.Context(), req.Email)
		if err != nil {
			respondError(c, http.StatusUnauthorized, badCredentials)
			return
		}

		// The lock is checked before the password, so guessing at a locked
		// account can neither be told apart from guessing wrong nor push the
		// window further out.
		if left := auth.LockedFor(user.LockedUntil, time.Now()); left > 0 {
			respondError(c, http.StatusUnauthorized, lockedMessage(left))
			return
		}

		if !auth.VerifyPassword(user.PasswordHash, req.Password) {
			respondError(c, http.StatusUnauthorized, recordFailedAttempt(c.Request.Context(), deps, user))
			return
		}

		clearThrottle(c.Request.Context(), deps, user)
		issueAuthResponse(c, deps, user, "logged in")
	}
}

// refreshHandler exchanges a valid refresh-token cookie for a fresh access
// token, without re-checking the password -- that's the entire point of a
// refresh token. It does not rotate the refresh token itself; session.go's
// 7-day TTL and the existing revoke-this-device/revoke-others Settings
// actions are the only ways it ever changes.
func refreshHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(deps.RefreshCookieName)
		if err != nil || cookie == "" {
			respondError(c, http.StatusUnauthorized, "no refresh token")
			return
		}
		refreshed, err := auth.RefreshSession(c.Request.Context(), deps.Queries, cookie)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrRefreshTokenReused):
				log.Printf("refresh: a replaced refresh token was reused; its session was revoked")
				// The session is gone, so the browser's newer token is dead too.
				clearRefreshCookie(c, deps)
			case errors.Is(err, auth.ErrInvalidRefreshToken):
				// Not cleared: this may be a request still carrying a token the
				// browser has since replaced, and clearing would take the new one.
			default:
				log.Printf("refresh: %v", err)
				respondError(c, http.StatusInternalServerError, "could not refresh your session")
				return
			}
			respondError(c, http.StatusUnauthorized, "refresh token expired or revoked, please log in again")
			return
		}
		user, err := deps.Queries.GetUserByID(c.Request.Context(), refreshed.UserID)
		if err != nil {
			clearRefreshCookie(c, deps)
			respondError(c, http.StatusUnauthorized, "refresh token expired or revoked, please log in again")
			return
		}

		accessToken, expiresAt, err := auth.IssueAccessToken(user.ID, deps.JWTSecret)
		if err != nil {
			log.Printf("refresh: issue access token: %v", err)
			respondError(c, http.StatusInternalServerError, "could not refresh your session")
			return
		}
		setRefreshCookie(c, deps, refreshed.Token, refreshed.ExpiresAt)
		respondSuccess(c, http.StatusOK, "session refreshed", authResponse{AccessToken: accessToken, ExpiresAt: expiresAt, User: newUserDTO(user)})
	}
}

// logoutHandler revokes the refresh token the request carries (if any) and
// clears the cookie. It never fails on a missing/already-invalid cookie --
// logging out an already-logged-out client is not an error.
func logoutHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cookie, err := c.Cookie(deps.RefreshCookieName); err == nil && cookie != "" {
			if err := auth.DeleteSession(c.Request.Context(), deps.Queries, cookie); err != nil {
				log.Printf("logout: delete session: %v", err)
			}
		}
		clearRefreshCookie(c, deps)
		respondSuccess[any](c, http.StatusOK, "logged out", nil)
	}
}

// meHandler returns the authenticated caller's own profile, for the client
// to call once on load (with whatever access token it has in memory) to
// learn who's signed in without a dedicated "whoami" field on every other
// response.
func meHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := UserID(c)
		if !ok {
			respondError(c, http.StatusUnauthorized, "not authenticated")
			return
		}
		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondSuccess(c, http.StatusOK, "user retrieved", newUserDTO(user))
	}
}

// issueAuthResponse mints both halves of a signed-in session for user --
// an access token returned in the body, and a refresh token set as an
// httpOnly cookie -- and writes the JSON response.
func issueAuthResponse(c *gin.Context, deps Deps, user sqlcgen.User, message string) {
	refreshToken, refreshExpiresAt, err := auth.CreateSession(c.Request.Context(), deps.Queries, user.ID, c.Request.UserAgent())
	if err != nil {
		log.Printf("issueAuthResponse: create session: %v", err)
		respondError(c, http.StatusInternalServerError, "could not create your session")
		return
	}
	setRefreshCookie(c, deps, refreshToken, refreshExpiresAt)

	accessToken, accessExpiresAt, err := auth.IssueAccessToken(user.ID, deps.JWTSecret)
	if err != nil {
		log.Printf("issueAuthResponse: issue access token: %v", err)
		respondError(c, http.StatusInternalServerError, "could not create your session")
		return
	}
	respondSuccess(c, http.StatusOK, message, authResponse{AccessToken: accessToken, ExpiresAt: accessExpiresAt, User: newUserDTO(user)})
}

// refreshCookieSameSite picks SameSite to match SecureCookies rather than
// hardcoding None: None requires Secure on every modern browser
// (Chrome rejects a SameSite=None cookie outright if Secure is missing,
// regardless of same-site-ness), and SecureCookies is false for local HTTP
// dev -- a hardcoded None silently dropped the cookie there entirely,
// which surfaced as every hard page reload bouncing back to /login despite
// a successful login. Lax is both valid without Secure and sufficient for
// local dev, where the Vite proxy makes the browser see one origin anyway;
// None+Secure is still what production (Vercel calling Render,
// genuinely cross-site) needs, and SecureCookies is true there.
func refreshCookieSameSite(secure bool) http.SameSite {
	if secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func setRefreshCookie(c *gin.Context, deps Deps, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     deps.RefreshCookieName,
		Value:    token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/api",
		SameSite: refreshCookieSameSite(deps.SecureCookies),
		Secure:   deps.SecureCookies,
	})
}

func clearRefreshCookie(c *gin.Context, deps Deps) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     deps.RefreshCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/api",
		SameSite: refreshCookieSameSite(deps.SecureCookies),
		Secure:   deps.SecureCookies,
	})
}

// recordFailedAttempt counts one wrong password against user and returns
// the message the client should show for it. Identical logic to
// handlers.recordFailedAttempt -- see its comments for the reasoning
// behind clearing a lapsed lock before counting and checking the lock
// before the password.
func recordFailedAttempt(ctx context.Context, deps Deps, user sqlcgen.User) string {
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

func clearThrottle(ctx context.Context, deps Deps, user sqlcgen.User) {
	if user.FailedLoginAttempts == 0 && !user.LockedUntil.Valid {
		return
	}
	if err := deps.Queries.ClearFailedLogins(ctx, user.ID); err != nil {
		log.Printf("login: clear failed attempts: %v", err)
	}
}

func lockedMessage(left time.Duration) string {
	if minutes := auth.LockMinutes(left); minutes != 1 {
		return fmt.Sprintf("too many failed attempts, try again in %d minutes", minutes)
	}
	return "too many failed attempts, try again in 1 minute"
}

// queueVerificationEmail mirrors handlers.queueVerificationEmail: best
// effort, never fails the request it was called from.
func queueVerificationEmail(ctx context.Context, deps Deps, userID int64, email string) {
	token, expiresAt, err := auth.CreateVerificationToken(ctx, deps.Queries, userID, email)
	if err != nil {
		log.Printf("verification email: create token: %v", err)
		return
	}
	if !deps.Mailer.Configured() {
		log.Printf("verification email: mailer not configured, skipping send to %s", email)
		return
	}
	link := deps.BaseURL + "/verify-email?token=" + token
	expiry := expiresAt.In(vietnamLocation).Format("15:04 on 2 Jan 2006")
	body := fmt.Sprintf(
		"Confirm this address to finish setting it up on your $pend account.\n\n"+
			"Verify your email: %s\n\n"+
			"This link expires at %s. If you didn't request this, you can ignore this email.",
		link, expiry)
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()
		if err := deps.Mailer.Send(sendCtx, email, "Verify your $pend email", body); err != nil {
			log.Printf("verification email: send: %v", err)
		}
	}()
}

package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// sessionDTO mirrors handlers.sessionView, except CreatedAt is a raw
// time.Time (not format.Timestamp's pre-formatted "02 Jan 2006, 15:04"
// string) -- the client formats its own timestamp, the same reasoning
// applied throughout this package's dashboard/category responses. Device
// stays server-resolved: format.DeviceLabel's User-Agent parsing has no
// client-side equivalent to push this one down to.
type sessionDTO struct {
	ID        string    `json:"id"`
	Device    string    `json:"device"`
	CreatedAt time.Time `json:"createdAt"`
	IsCurrent bool      `json:"isCurrent"`
}

// settingsResponse is GET /api/v1/settings' body: the profile fields every
// form on the HTML settings page pre-fills itself with, plus the active
// sessions list. There is no Saved/error-message field here the way
// handlers.settingsView has -- that existed only to survive a
// POST-redirect-GET round trip; a JSON mutation response answers for
// itself (the envelope's success and message) without needing the next GET to
// carry the verdict.
type settingsResponse struct {
	Name         string       `json:"name"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	PendingEmail string       `json:"pendingEmail,omitempty"`
	Sessions     []sessionDTO `json:"sessions"`
}

func settingsHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load settings")
			return
		}

		currentToken, _ := c.Cookie(deps.RefreshCookieName)
		sessions, err := deps.Queries.ListSessionsForUser(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load settings")
			return
		}
		dtos := make([]sessionDTO, 0, len(sessions))
		for _, s := range sessions {
			dtos = append(dtos, sessionDTO{
				ID:        s.ID,
				Device:    format.DeviceLabel(s.UserAgent.String),
				CreatedAt: s.CreatedAt.Time,
				IsCurrent: s.ID == currentToken,
			})
		}

		respondOK(c, http.StatusOK, "settings retrieved", settingsResponse{
			Name: user.Name, Username: user.Username, Email: user.Email,
			PendingEmail: user.PendingEmail.String, Sessions: dtos,
		})
	}
}

type updateProfileRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

func updateProfileHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req updateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		name := strings.TrimSpace(req.Name)
		username := strings.ToLower(strings.TrimSpace(req.Username))

		if name == "" {
			respondError(c, http.StatusBadRequest, "please enter your name")
			return
		}
		if !usernamePattern.MatchString(username) {
			respondError(c, http.StatusBadRequest, "username must be 3-20 characters: lowercase letters, numbers, or underscores, starting with a letter")
			return
		}

		err := deps.Queries.UpdateUserProfile(c.Request.Context(), sqlcgen.UpdateUserProfileParams{
			ID: userID, Name: name, Username: username,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				respondError(c, http.StatusConflict, "that username is already taken")
				return
			}
			log.Printf("update profile: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update profile")
			return
		}
		respondOK[any](c, http.StatusOK, "profile updated", nil)
	}
}

type updateEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"currentPassword"`
}

// updateEmailHandler mirrors handlers.updateEmailHandler exactly, including
// its comment's reasoning: it only stages pending_email and never touches
// users.email directly -- ApplyVerifiedEmail (auth_email_verification.go)
// does that once the emailed link is visited, so a mistyped address can
// never cost the owner the one they can still be reached at.
func updateEmailHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req updateEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		email := strings.TrimSpace(req.Email)

		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not update email")
			return
		}
		if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
			respondError(c, http.StatusUnauthorized, "that current password is not correct")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			respondError(c, http.StatusBadRequest, "that email address is not valid")
			return
		}
		if existing, err := deps.Queries.GetUserByEmail(c.Request.Context(), email); err == nil && existing.ID != userID {
			respondError(c, http.StatusConflict, "that email is already registered")
			return
		}

		if err := deps.Queries.SetPendingEmail(c.Request.Context(), sqlcgen.SetPendingEmailParams{
			ID: userID, PendingEmail: pgtype.Text{String: email, Valid: true},
		}); err != nil {
			log.Printf("update email: set pending: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update email")
			return
		}

		queueVerificationEmail(c.Request.Context(), deps, userID, email)
		respondOK[any](c, http.StatusOK, "check your new inbox to confirm the change", nil)
	}
}

// resendVerificationHandler re-queues the verification email for whichever
// address needs confirming -- pending_email if an email change is staged,
// otherwise the account's own email if it isn't verified yet.
func resendVerificationHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not resend verification email")
			return
		}
		target := user.Email
		if user.PendingEmail.Valid {
			target = user.PendingEmail.String
		} else if user.EmailVerified {
			respondError(c, http.StatusBadRequest, "your email is already verified")
			return
		}
		queueVerificationEmail(c.Request.Context(), deps, userID, target)
		respondOK[any](c, http.StatusOK, "verification email sent", nil)
	}
}

type updatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword"`
	NewPassword        string `json:"newPassword"`
	NewPasswordConfirm string `json:"newPasswordConfirm"`
}

func updatePasswordHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req updatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}

		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not update password")
			return
		}
		if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
			respondError(c, http.StatusUnauthorized, "that current password is not correct")
			return
		}
		if len([]rune(req.NewPassword)) < 8 {
			respondError(c, http.StatusBadRequest, "new password must be at least 8 characters")
			return
		}
		if req.NewPassword != req.NewPasswordConfirm {
			respondError(c, http.StatusBadRequest, "the two new passwords do not match")
			return
		}
		if req.NewPassword == req.CurrentPassword {
			respondError(c, http.StatusBadRequest, "the new password must be different from the current one")
			return
		}

		hash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("update password: hash: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update password")
			return
		}
		if err := deps.Queries.UpdateUserPassword(c.Request.Context(), sqlcgen.UpdateUserPasswordParams{
			ID: userID, PasswordHash: hash,
		}); err != nil {
			log.Printf("update password: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update password")
			return
		}

		// Every *other* refresh token is revoked, the same action a
		// password change takes on the HTML side -- the current one
		// (read off the request's own refresh-token cookie) survives, so
		// whoever just changed the password isn't logged out by it.
		if cookie, err := c.Cookie(deps.RefreshCookieName); err == nil {
			if err := deps.Queries.DeleteOtherSessionsForUser(c.Request.Context(), sqlcgen.DeleteOtherSessionsForUserParams{
				UserID: userID, ID: cookie,
			}); err != nil {
				log.Printf("update password: delete other sessions: %v", err)
			}
		}
		respondOK[any](c, http.StatusOK, "password updated", nil)
	}
}

func revokeSessionHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		sessionID := c.Param("id")
		if err := deps.Queries.DeleteSessionForUser(c.Request.Context(), sqlcgen.DeleteSessionForUserParams{
			ID: sessionID, UserID: userID,
		}); err != nil {
			log.Printf("revoke session: %v", err)
			respondError(c, http.StatusInternalServerError, "could not revoke session")
			return
		}
		respondOK[any](c, http.StatusOK, "session revoked", nil)
	}
}

func revokeOtherSessionsHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		cookie, err := c.Cookie(deps.RefreshCookieName)
		if err != nil {
			respondError(c, http.StatusBadRequest, "no refresh token on this request")
			return
		}
		if err := deps.Queries.DeleteOtherSessionsForUser(c.Request.Context(), sqlcgen.DeleteOtherSessionsForUserParams{
			UserID: userID, ID: cookie,
		}); err != nil {
			log.Printf("revoke other sessions: %v", err)
			respondError(c, http.StatusInternalServerError, "could not revoke other sessions")
			return
		}
		respondOK[any](c, http.StatusOK, "other sessions revoked", nil)
	}
}

type deleteAccountRequest struct {
	CurrentPassword string `json:"currentPassword"`
}

// deleteAccountHandler mirrors handlers.deleteAccountHandler/deleteAccount
// exactly -- see .claude/rules/account-deletion.md for the full reasoning
// (hard delete, no grace period, explicit transaction order, never touches
// shared defaults). On success it also revokes the refresh-token cookie
// the request carried, the JSON equivalent of handlers.go's
// clearSessionCookie -- though since deleteAccount's transaction removes
// every session row via the cascade, the cookie value is already an
// orphaned token; clearing it just stops the browser resending it.
func deleteAccountHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req deleteAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}

		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not delete account")
			return
		}
		if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
			respondError(c, http.StatusUnauthorized, "that password is not correct")
			return
		}

		if err := deleteAccount(c.Request.Context(), deps, userID); err != nil {
			log.Printf("delete account: %v", err)
			respondError(c, http.StatusInternalServerError, "could not delete account")
			return
		}
		clearRefreshCookie(c, deps)
		respondOK[any](c, http.StatusOK, "account deleted", nil)
	}
}

// deleteAccount mirrors handlers.deleteAccount exactly -- see
// .claude/rules/account-deletion.md's "Why" section for why the order is
// spelled out explicitly rather than left to users' ON DELETE CASCADE.
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
	if err := qtx.DeletePersonalCategoriesForUser(ctx, pgval.Int64(userID)); err != nil {
		return fmt.Errorf("delete categories: %w", err)
	}
	if err := qtx.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return tx.Commit(ctx)
}

// --- theme ---

// Mirrors handlers/settings_theme.go's three accepted values exactly (see
// its comment for why "auto" is never resolved server-side).
const (
	themeAuto  = "auto"
	themeLight = "light"
	themeDark  = "dark"
)

func validTheme(v string) bool {
	return v == themeAuto || v == themeLight || v == themeDark
}

type updateThemeRequest struct {
	Theme string `json:"theme"`
}

func updateThemeHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req updateThemeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		if !validTheme(req.Theme) {
			respondError(c, http.StatusBadRequest, "invalid theme")
			return
		}
		if err := deps.Queries.UpdateUserTheme(c.Request.Context(), sqlcgen.UpdateUserThemeParams{
			ID: userID, Theme: req.Theme,
		}); err != nil {
			log.Printf("update theme: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update theme")
			return
		}
		respondOK[any](c, http.StatusOK, "theme updated", nil)
	}
}

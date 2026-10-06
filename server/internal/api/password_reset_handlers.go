package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

// sendTimeout mirrors handlers.sendTimeout: bounds how long a single
// background reset/verification email send may run, so a slow or
// unreachable mail provider can never leak a goroutine.
const sendTimeout = 10 * time.Second

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPasswordHandler mirrors handlers.forgotPasswordPage's POST case
// exactly: the response is identical whether or not the submitted email
// matches an account (a token is only created and an email only sent when
// it does), so this endpoint can't be used to enumerate registered
// addresses.
func forgotPasswordHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req forgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		email := strings.TrimSpace(req.Email)

		if user, err := deps.Queries.GetUserByEmail(c.Request.Context(), email); err == nil {
			queueResetEmail(c.Request.Context(), deps, user)
		}
		respondOK[any](c, http.StatusOK, "if that email is registered, a reset link is on its way", nil)
	}
}

// queueResetEmail mirrors handlers.queueResetEmail exactly: a fast local
// token write done synchronously, then the actual send handed to a
// background goroutine with its own timeout, independent of the request's
// own (already-closed-by-the-time-it-would-matter) context.
func queueResetEmail(ctx context.Context, deps Deps, user sqlcgen.User) {
	token, expiresAt, err := auth.CreateResetToken(ctx, deps.Queries, user.ID)
	if err != nil {
		log.Printf("forgot password: create token: %v", err)
		return
	}
	if !deps.Mailer.Configured() {
		log.Printf("forgot password: mailer not configured, skipping send to %s", user.Email)
		return
	}
	link := deps.BaseURL + "/reset-password?token=" + token
	expiry := expiresAt.In(vietnamLocation).Format("15:04 on 2 Jan 2006")
	body := fmt.Sprintf(
		"Someone requested a password reset for your $pend account.\n\n"+
			"Reset your password: %s\n\n"+
			"This link expires at %s. If you didn't request this, you can ignore this email.",
		link, expiry)
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()
		if err := deps.Mailer.Send(sendCtx, user.Email, "Reset your $pend password", body); err != nil {
			log.Printf("forgot password: send email: %v", err)
		}
	}()
}

// checkResetTokenHandler lets the client validate a token as soon as the
// reset-password page loads (before the visitor types anything), mirroring
// handlers.resetPasswordPage's GET case -- an expired or already-spent
// link shows as invalid immediately rather than after a submit.
func checkResetTokenHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if _, err := auth.ValidateResetToken(c.Request.Context(), deps.Queries, token); err != nil {
			respondError(c, http.StatusNotFound, "that reset link is invalid or has expired")
			return
		}
		respondOK[any](c, http.StatusOK, "reset link is valid", nil)
	}
}

type resetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

// resetPasswordHandler mirrors handlers.resetPasswordPage's POST case
// exactly, including signing the visitor in on success: unlike
// updatePasswordHandler there is no current session to spare (the visitor
// arrived signed out), so every device is logged out and the fresh
// session issueAuthResponse starts is what signs them back in.
func resetPasswordHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req resetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}

		userID, err := auth.ValidateResetToken(c.Request.Context(), deps.Queries, req.Token)
		if err != nil {
			respondError(c, http.StatusNotFound, "that reset link is invalid or has expired")
			return
		}
		if len([]rune(req.Password)) < 8 {
			respondError(c, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		if req.Password != req.PasswordConfirm {
			respondError(c, http.StatusBadRequest, "the two passwords do not match")
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			log.Printf("reset password: hash: %v", err)
			respondError(c, http.StatusInternalServerError, "could not reset password")
			return
		}
		if err := deps.Queries.UpdateUserPassword(c.Request.Context(), sqlcgen.UpdateUserPasswordParams{
			ID: userID, PasswordHash: hash,
		}); err != nil {
			log.Printf("reset password: update: %v", err)
			respondError(c, http.StatusInternalServerError, "could not reset password")
			return
		}
		// A reset is the way out of a login lock that doesn't involve
		// waiting the window out.
		if err := deps.Queries.ClearFailedLogins(c.Request.Context(), userID); err != nil {
			log.Printf("reset password: clear failed attempts: %v", err)
		}
		if err := auth.ConsumeResetToken(c.Request.Context(), deps.Queries, req.Token); err != nil {
			log.Printf("reset password: consume token: %v", err)
		}
		if err := deps.Queries.DeleteSessionsForUser(c.Request.Context(), userID); err != nil {
			log.Printf("reset password: delete sessions: %v", err)
		}

		user, err := deps.Queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not reset password")
			return
		}
		issueAuthResponse(c, deps, user, "password reset")
	}
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// verifyEmailResponse's two booleans mirror handlers.verifyEmailView
// exactly: Verified is success, Conflict is the address having been
// claimed by another account in the meantime, and neither set (false,
// false) covers a link that was expired, already spent, or never valid --
// handlers.go's template {{else}} branch, here just the response's zero
// value.
type verifyEmailResponse struct {
	Verified bool `json:"verified"`
	Conflict bool `json:"conflict"`
}

// verifyEmailHandler mirrors handlers.verifyEmailPage exactly. It is
// unauthenticated on purpose, same as the HTML route: the browser opening
// the link is often not the one the visitor is signed in on, and the
// token itself is what proves the request is legitimate.
func verifyEmailHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req verifyEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}

		userID, email, err := auth.ValidateVerificationToken(c.Request.Context(), deps.Queries, req.Token)
		if err != nil {
			respondOK(c, http.StatusOK, "that verification link is invalid or has expired", verifyEmailResponse{})
			return
		}

		if err := deps.Queries.ApplyVerifiedEmail(c.Request.Context(), sqlcgen.ApplyVerifiedEmailParams{
			ID: userID, Email: email,
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				respondOK(c, http.StatusOK, "that email is already used by another account", verifyEmailResponse{Conflict: true})
				return
			}
			log.Printf("verify email: apply: %v", err)
			respondError(c, http.StatusInternalServerError, "could not verify email")
			return
		}
		if err := auth.ConsumeVerificationToken(c.Request.Context(), deps.Queries, req.Token); err != nil {
			log.Printf("verify email: consume token: %v", err)
		}
		respondOK(c, http.StatusOK, "email verified", verifyEmailResponse{Verified: true})
	}
}

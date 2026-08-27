package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
)

// queueVerificationEmail issues a verification token proving email belongs
// to userID -- a fast local write, done synchronously while ctx is still
// valid -- and hands the actual send to a background goroutine, the same
// split queueResetEmail uses and for the same reason: the caller (a
// registration or a settings POST) must not wait on a mail provider that
// might be slow or unreachable, and the request's own context would be
// canceled the moment the handler returns.
func queueVerificationEmail(ctx context.Context, deps Deps, userID int64, email string) {
	token, expiresAt, err := deps.EmailVerifications.CreateVerificationToken(ctx, userID, email)
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

// verifyEmailPage handles the link a verification email points at. It is
// public rather than behind auth.RequireAuth: the browser that opens it is
// often not the one the visitor is signed in on, and the token itself is
// what proves the request is legitimate, the same way /reset-password's
// token does.
func verifyEmailPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")

		userID, email, err := deps.EmailVerifications.ValidateVerificationToken(r.Context(), token)
		if err != nil {
			render(w, r, deps, "verify_email", "", map[string]any{"Invalid": true})
			return
		}

		if err := deps.Queries.ApplyVerifiedEmail(r.Context(), sqlcgen.ApplyVerifiedEmailParams{
			ID: userID, Email: email,
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				render(w, r, deps, "verify_email", "", map[string]any{"Conflict": true})
				return
			}
			log.Printf("verify email: apply: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.EmailVerifications.ConsumeVerificationToken(r.Context(), token); err != nil {
			log.Printf("verify email: consume token: %v", err)
		}

		render(w, r, deps, "verify_email", "", map[string]any{"Verified": true})
	}
}

// resendVerificationHandler reissues a link for whichever address is
// currently unconfirmed: PendingEmail if a change is in flight, otherwise
// the account's own Email for a signup that never got verified.
func resendVerificationHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		user, err := deps.Queries.GetUserByID(r.Context(), userID)
		if err != nil {
			log.Printf("resend verification: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		email := user.Email
		if user.PendingEmail.Valid {
			email = user.PendingEmail.String
		}
		queueVerificationEmail(r.Context(), deps, userID, email)

		http.Redirect(w, r, "/settings?saved=verification-sent", http.StatusSeeOther)
	}
}

package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

// renderForgotPasswordFragmentOrPage mirrors renderAuthFragmentOrPage: an
// htmx submit gets back just the card body to swap into #forgot-password-card,
// a direct navigation gets the full page shell.
func renderForgotPasswordFragmentOrPage(w http.ResponseWriter, r *http.Request, deps Deps, data map[string]any) {
	if isFragmentRequest(r) {
		renderNamed(w, r, deps, "forgot_password", "forgot_password_card_body", "", data)
		return
	}
	render(w, r, deps, "forgot_password", "", data)
}

// forgotPasswordPage handles both the request-a-link form and its
// submission. The response is deliberately identical whether or not the
// submitted email matches an account -- a token is only created and an
// email only sent when it does -- so this endpoint can't be used to find
// out which addresses are registered.
func forgotPasswordPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderForgotPasswordFragmentOrPage(w, r, deps, map[string]any{})
			return
		}

		email := strings.TrimSpace(r.FormValue("email"))

		if user, err := deps.Queries.GetUserByEmail(r.Context(), email); err == nil {
			sendResetEmail(r, deps, user)
		}

		renderForgotPasswordFragmentOrPage(w, r, deps, map[string]any{"Email": email, "Sent": true})
	}
}

// sendResetEmail issues a reset token for user and mails the link it
// authorizes. Failures are logged rather than surfaced to the caller: the
// forgot-password response never reveals whether this step even ran, let
// alone whether it succeeded.
func sendResetEmail(r *http.Request, deps Deps, user sqlcgen.User) {
	token, expiresAt, err := deps.PasswordResets.CreateResetToken(r.Context(), user.ID)
	if err != nil {
		log.Printf("forgot password: create token: %v", err)
		return
	}

	if !deps.Mailer.Configured() {
		log.Printf("forgot password: SMTP not configured, skipping send to %s", user.Email)
		return
	}

	link := deps.BaseURL + "/reset-password?token=" + token
	expiry := expiresAt.In(vietnamLocation).Format("15:04 on 2 Jan 2006")
	body := fmt.Sprintf(
		"Someone requested a password reset for your $pend account.\n\n"+
			"Reset your password: %s\n\n"+
			"This link expires at %s. If you didn't request this, you can ignore this email.",
		link, expiry)

	if err := deps.Mailer.Send(user.Email, "Reset your $pend password", body); err != nil {
		log.Printf("forgot password: send email: %v", err)
	}
}

// renderResetPasswordFragmentOrPage mirrors renderForgotPasswordFragmentOrPage
// for the /reset-password card.
func renderResetPasswordFragmentOrPage(w http.ResponseWriter, r *http.Request, deps Deps, data map[string]any) {
	if isFragmentRequest(r) {
		renderNamed(w, r, deps, "reset_password", "reset_password_card_body", "", data)
		return
	}
	render(w, r, deps, "reset_password", "", data)
}

// resetPasswordPage handles both the landing GET (which validates the token
// up front so a dead link shows as invalid before the visitor types
// anything) and the new-password submission.
func resetPasswordPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			token := r.URL.Query().Get("token")
			if _, err := deps.PasswordResets.ValidateResetToken(r.Context(), token); err != nil {
				renderResetPasswordFragmentOrPage(w, r, deps, map[string]any{"Invalid": true})
				return
			}
			renderResetPasswordFragmentOrPage(w, r, deps, map[string]any{"Token": token})
			return
		}

		token := r.FormValue("token")
		next := r.FormValue("password")
		confirm := r.FormValue("password_confirm")

		userID, err := deps.PasswordResets.ValidateResetToken(r.Context(), token)
		if err != nil {
			renderResetPasswordFragmentOrPage(w, r, deps, map[string]any{"Invalid": true})
			return
		}

		fail := func(msg string) {
			renderResetPasswordFragmentOrPage(w, r, deps, map[string]any{"Token": token, "Error": msg})
		}

		if len([]rune(next)) < 8 {
			fail("Password must be at least 8 characters.")
			return
		}
		if next != confirm {
			fail("The two passwords do not match.")
			return
		}

		hash, err := auth.HashPassword(next)
		if err != nil {
			log.Printf("reset password: hash: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.Queries.UpdateUserPassword(r.Context(), sqlcgen.UpdateUserPasswordParams{
			ID: userID, PasswordHash: hash,
		}); err != nil {
			log.Printf("reset password: update: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.PasswordResets.ConsumeResetToken(r.Context(), token); err != nil {
			log.Printf("reset password: consume token: %v", err)
		}
		// No current session to spare here, unlike updatePasswordHandler --
		// the visitor arrived signed out -- so every device is logged out
		// and the fresh session started below is what signs them back in.
		if err := deps.Queries.DeleteSessionsForUser(r.Context(), userID); err != nil {
			log.Printf("reset password: delete sessions: %v", err)
		}

		startSession(w, r, deps, userID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

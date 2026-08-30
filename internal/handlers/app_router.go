package handlers

import (
	"net/http"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/csrf"
	"expensetracker/internal/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter constructs the application's HTTP handler with all routes registered.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// The Worker webhook is the one route with no browser behind it, so it
	// has no cookie to double-submit and csrf.Middleware would reject every
	// email with 403. The exemption is expressed here, in the router, rather
	// than inside internal/csrf, which has no business knowing route paths.
	r.Use(csrfExcept(csrf.Middleware(deps.SecureCookies), isInboxWebhookRequest))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Public: the caller is the Cloudflare Email Worker, not a browser
	// carrying a session cookie.
	r.Post("/inbox/{token}", inboxWebhookHandler(deps))

	// Public: the login page needs the stylesheet and app.js before anyone
	// is authenticated.
	r.Handle(web.StaticPrefix+"*", web.StaticHandler())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	r.Get("/register", registerPage(deps))
	r.Post("/register", registerPage(deps))
	r.Get("/login", loginPage(deps))
	r.Post("/login", loginPage(deps))
	r.Post("/logout", logoutHandler(deps))
	r.Get("/forgot-password", forgotPasswordPage(deps))
	r.Post("/forgot-password", forgotPasswordPage(deps))
	r.Get("/reset-password", resetPasswordPage(deps))
	r.Post("/reset-password", resetPasswordPage(deps))
	r.Get("/verify-email", verifyEmailPage(deps))

	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireAuth(deps.Sessions, deps.CookieName))
		pr.Get("/categories", categoriesPage(deps))
		pr.Post("/categories", categoriesPage(deps))
		pr.Patch("/categories/{id}/color", updateCategoryColorHandler(deps))
		pr.Get("/categories/{id}/edit", editCategoryHandler(deps))
		pr.Get("/categories/{id}/view", viewCategoryRowHandler(deps))
		pr.Patch("/categories/{id}/name", updateCategoryNameHandler(deps))
		pr.Delete("/categories/{id}", deleteCategoryHandler(deps))
		pr.Get("/transactions", transactionsPage(deps))
		pr.Post("/transactions", transactionsPage(deps))
		pr.Get("/transactions/export", exportTransactionsHandler(deps))
		pr.Get("/transactions/import", importPage(deps))
		pr.Post("/transactions/import", importHandler(deps))
		pr.Get("/transactions/category-options", categoryPickerHandler(deps, "category_options"))
		pr.Get("/transactions/category-chips", categoryPickerHandler(deps, "category_chips"))
		pr.Get("/transactions/{id}/edit", editTransactionRowHandler(deps))
		pr.Get("/transactions/{id}/view", viewTransactionRowHandler(deps))
		pr.Get("/transactions/{id}/delete-confirm", deleteConfirmTransactionHandler(deps))
		pr.Patch("/transactions/{id}", updateTransactionHandler(deps))
		pr.Delete("/transactions/{id}", deleteTransactionHandler(deps))
		pr.Get("/dashboard", dashboardPage(deps))
		pr.Get("/settings", settingsPage(deps))
		pr.Post("/settings/profile", updateProfileHandler(deps))
		pr.Post("/settings/email", updateEmailHandler(deps))
		pr.Post("/resend-verification", resendVerificationHandler(deps))
		pr.Post("/settings/password", updatePasswordHandler(deps))
		pr.Post("/settings/delete", deleteAccountHandler(deps))
		pr.Post("/settings/sessions/revoke", revokeSessionHandler(deps))
		pr.Post("/settings/sessions/revoke-others", revokeOtherSessionsHandler(deps))
		pr.Post("/settings/inbox/enable", enableInboxHandler(deps))
		pr.Post("/settings/inbox/disable", disableInboxHandler(deps))
		pr.Post("/settings/inbox/retry", retryFailedEmailsHandler(deps))
		pr.Put("/settings/theme", updateThemeHandler(deps))
	})

	return r
}

// csrfExcept applies mw to every request except those for which except
// reports true.
func csrfExcept(mw func(http.Handler) http.Handler, except func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		guarded := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if except(r) {
				next.ServeHTTP(w, r)
				return
			}
			guarded.ServeHTTP(w, r)
		})
	}
}

// isInboxWebhookRequest reports whether r is exactly POST /inbox/{token} --
// the one route with no browser behind it, so it has no CSRF cookie to
// double-submit.
//
// The match is deliberately narrow rather than a path-prefix test: method
// POST, plus exactly one non-empty path segment after "/inbox/" with no
// further "/". A future browser-facing page added under /inbox/ -- an inbox
// log view, say -- would otherwise silently inherit this exemption merely by
// sharing the prefix, with nothing at that new call site to prompt a second
// look. Resist loosening this back to strings.HasPrefix.
func isInboxWebhookRequest(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	rest := strings.TrimPrefix(r.URL.Path, "/inbox/")
	if rest == r.URL.Path {
		// No "/inbox/" prefix at all.
		return false
	}
	return rest != "" && !strings.Contains(rest, "/")
}

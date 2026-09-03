package handlers

import (
	"html/template"

	"expensetracker/internal/classify"
	"expensetracker/internal/mailer"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds shared dependencies for handlers.
//
// Templates is keyed by page name (e.g. "auth", "categories"). Each entry
// is a *template.Template built from layout.html plus that page's own
// template file(s), so that every template set has only a single
// {{define "content"}} block in scope. This avoids a collision that would
// occur if all page templates were parsed together into one shared
// *template.Template: Go's html/template registers "content" as a single
// global name per template set, so the last-parsed page's block would win
// for every page.
type Deps struct {
	DB      *pgxpool.Pool
	Queries *sqlcgen.Queries
	Mailer  *mailer.Mailer
	// Classifier resolves a category for a bank-email transaction when no
	// category_hints row fits (see resolveCategoryForNotice in
	// internal/inboxproc). Nil is treated the same as an unconfigured
	// Classifier -- see that function's own guard -- but every real
	// construction path (main.go, and the handler test helpers that
	// exercise email processing) sets it via classify.New, the same way
	// Mailer is always constructed even with an empty Config.
	Classifier *classify.Classifier
	Templates  map[string]*template.Template
	CookieName string
	// SecureCookies gates the Secure attribute on the session and CSRF
	// cookies; see internal/config.Config.SecureCookies for how it's
	// populated.
	SecureCookies bool
	// BaseURL is used to build the absolute link a password-reset email
	// points at; see internal/config.Config.BaseURL.
	BaseURL string
	// InboundDomain and InboundWebhookSecret configure the email ingestion
	// path; see internal/config.Config for what an empty value means.
	InboundDomain        string
	InboundWebhookSecret string
}

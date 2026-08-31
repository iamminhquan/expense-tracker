package handlers

import (
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// recentEmailsShown caps the settings card's list of forwarded emails. It is
// a status window, not an archive: everything older stays in the table, and
// the retry button (scoped to 'failed' regardless of what is shown) covers
// every failed row whether or not it made this list.
const recentEmailsShown = 10

// recentEmailView is one row of that list.
type recentEmailView struct {
	Subject    string
	ReceivedAt string
	Status     string
	Reason     string
}

// enableInboxHandler switches email tracking on, and switches it to a new
// address if it was already on. Regenerating is the revocation path: an
// address that leaked is retired the moment a new one is issued, because the
// webhook resolves the token against this column and nothing else -- an
// email already in flight to the old address arrives after this UPDATE and
// finds no user row whose inbox_token matches it, so
// GetUserByInboxToken (inbox_webhook.go) answers 404 rather than
// misattributing it to whichever account holds the new token.
func enableInboxHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		token, err := inbound.NewToken()
		if err != nil {
			serverError(w, "enable inbox: new token", err)
			return
		}
		if err := deps.Queries.SetInboxToken(r.Context(), sqlcgen.SetInboxTokenParams{
			ID: userID, InboxToken: pgtype.Text{String: token, Valid: true},
		}); err != nil {
			serverError(w, "enable inbox", err)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-enabled", http.StatusSeeOther)
	}
}

// disableInboxHandler switches email tracking off. Stored email is left
// alone: it is the owner's own history, and turning the address off is not a
// request to erase what already arrived.
func disableInboxHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		if err := deps.Queries.SetInboxToken(r.Context(), sqlcgen.SetInboxTokenParams{
			ID: userID, InboxToken: pgtype.Text{},
		}); err != nil {
			serverError(w, "disable inbox", err)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-disabled", http.StatusSeeOther)
	}
}

// retryFailedEmailsHandler puts every failed email back in the queue. This
// button is the entire reason email is stored raw before it is read: a parser
// fix is worth nothing if the messages it would now understand are gone.
func retryFailedEmailsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		if err := deps.Queries.RequeueFailedBankEmails(r.Context(), userID); err != nil {
			serverError(w, "retry failed emails", err)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-retried", http.StatusSeeOther)
	}
}

// addInboxSettings fills in the Email tracking card's values. It leaves
// InboxAvailable false when no domain is configured, which is what makes
// the card disappear rather than render an address nobody can send to.
func addInboxSettings(r *http.Request, deps Deps, userID int64, token pgtype.Text, data *settingsView) error {
	if deps.InboundDomain == "" {
		return nil
	}

	data.InboxAvailable = true
	data.InboxEnabled = token.Valid && token.String != ""
	if token.Valid && token.String != "" {
		data.InboxAddress = token.String + "@" + deps.InboundDomain
	}

	rows, err := deps.Queries.ListRecentBankEmails(r.Context(), sqlcgen.ListRecentBankEmailsParams{
		UserID: userID, Limit: recentEmailsShown,
	})
	if err != nil {
		return err
	}
	views := make([]recentEmailView, 0, len(rows))
	for _, row := range rows {
		views = append(views, recentEmailView{
			Subject:    row.Subject,
			ReceivedAt: format.Timestamp(row.ReceivedAt, vietnamLocation),
			Status:     row.Status,
			Reason:     row.FailureReason,
		})
	}
	data.InboxRecent = views

	// The retry button is scoped to 'failed' rows -- it must stay hidden
	// when there are none, even if the list above is showing only
	// pending/imported rows because nothing has failed recently. A second,
	// separate query rather than deriving this from views above: the two
	// have to agree independently of which statuses happen to fill the
	// capped list.
	failed, err := deps.Queries.ListRecentFailedBankEmails(r.Context(), sqlcgen.ListRecentFailedBankEmailsParams{
		UserID: userID, Limit: 1,
	})
	if err != nil {
		return err
	}
	data.InboxHasFailed = len(failed) > 0
	return nil
}

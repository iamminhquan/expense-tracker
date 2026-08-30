package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// bankSenders are the addresses a balance-change notice may legitimately
// arrive from. This is the third auth layer and the load-bearing one:
// without it, anyone who learns a forwarding address could post invented
// email and put invented transactions straight into someone's books.
var bankSenders = []string{
	// Confirmed against a real notice: MB sends from mbebanking@mbbank.com.vn.
	// The first guess here was @mb.com.vn, which matched nothing -- a real
	// transfer was what settled it, which is why ingestion shipped before the
	// parser.
	"@mbbank.com.vn",
	// Still guesses. No TPBank notice has arrived yet, so whichever of these
	// is wrong will show up the same way MB's did: the message lands with
	// status 'ignored' and its real from_address readable in bank_emails.
	"@tpb.com.vn",
	"@tpbank.com.vn",
}

// isKnownBankSender reports whether from belongs to a bank this app reads.
// Matching is on the domain suffix rather than the whole address, because
// banks send from several local parts and change them without notice.
func isKnownBankSender(from string) bool {
	addr := strings.ToLower(strings.TrimSpace(from))
	if i := strings.LastIndex(addr, "<"); i >= 0 {
		addr = strings.TrimSuffix(addr[i+1:], ">")
	}
	for _, suffix := range bankSenders {
		if strings.HasSuffix(addr, suffix) {
			return true
		}
	}
	return false
}

// inboxWebhookHandler receives one forwarded email from the Cloudflare Email
// Worker, stores it raw, and answers 200 without doing anything else. Parsing
// it into a transaction is a later slice's job and deliberately not this
// request's: the Worker is waiting on this response, and a parser that is
// still wrong should not be able to lose an email.
//
// Three layers decide whether the caller is who they claim to be: the token
// in the path names an account, the HMAC proves the body came from our own
// Worker, and the sender address proves the message came from a bank. An
// email failing only the third is still stored, as 'ignored' -- keeping it
// costs nothing and makes "why did nothing happen" answerable.
func inboxWebhookHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The request cap sits well above MaxBodyBytes on purpose: that
		// constant bounds only the plain-text body kept in bank_emails, while
		// this reads the whole JSON envelope -- from/to/subject plus escaping
		// overhead around a text field that is itself allowed to reach
		// MaxBodyBytes. The multiplier leaves real headroom rather than
		// rejecting a legitimate envelope right at the boundary.
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2*(inbound.MaxBodyBytes+inbound.MaxRawBytes)))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}
			// Anything else here is a client that hung up mid-upload or sent
			// a malformed request, not a size problem.
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if !inbound.Verify(deps.InboundWebhookSecret, body, r.Header.Get(inbound.SignatureHeader)) {
			http.Error(w, "bad signature", http.StatusForbidden)
			return
		}

		user, err := deps.Queries.GetUserByInboxToken(r.Context(), pgText(chi.URLParam(r, "token")))
		if errors.Is(err, pgx.ErrNoRows) {
			// No account owns this address -- most likely one whose owner
			// turned tracking off and regenerated their token.
			http.Error(w, "unknown inbox", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("inbox webhook: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		payload, err := inbound.ParsePayload(body)
		if err != nil {
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}

		messageID := strings.TrimSpace(payload.MessageID)
		if messageID == "" {
			// Not every forwarder sets Message-ID, and the unique index that
			// stops a duplicate still needs a value.
			messageID = inbound.Fingerprint(payload)
		}

		status, reason := "pending", ""
		if !isKnownBankSender(payload.From) {
			status, reason = "ignored", "sender is not a bank this app reads"
		}

		_, err = deps.Queries.CreateBankEmail(r.Context(), sqlcgen.CreateBankEmailParams{
			UserID:        user.ID,
			MessageID:     messageID,
			FromAddress:   payload.From,
			Subject:       payload.Subject,
			Body:          inbound.TruncateBody(payload.Text),
			RawBody:       inbound.TruncateRaw(payload.Raw),
			Status:        status,
			FailureReason: reason,
		})
		// pgx.ErrNoRows here is the ON CONFLICT DO NOTHING firing: the same
		// message already arrived once. That is a success, not a failure --
		// answering anything else would make the Worker retry forever.
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("inbox webhook: store email: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

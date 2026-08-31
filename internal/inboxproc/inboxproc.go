// Package inboxproc turns the bank email a user's inbox has already
// received into transactions.
//
// It is the writing half of email ingestion: internal/bankmail reads a
// message into a Notice and touches nothing, internal/handlers accepts the
// Worker's POST and stores the raw message, and this package is what runs
// afterwards, on its own goroutine, deciding what each stored message
// means for the ledger. It lived in internal/handlers until it was clear
// that nothing here answers a request -- there is no ResponseWriter on any
// path through it, and the only two things it ever needed from Deps were
// the queries and the classifier.
package inboxproc

import (
	"context"
	"errors"
	"fmt"
	"log"

	"expensetracker/internal/bankmail"
	"expensetracker/internal/classify"
	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
	"expensetracker/internal/txnrule"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// A Processor reads stored bank email for one deployment's database.
//
// It holds the two things this work needs and nothing else: the queries,
// and the classifier consulted when no remembered hint fits. A nil
// classifier is the ordinary state of an account that never set
// GEMINI_API_KEY and is handled as such -- see classifyAndRememberCategory.
type Processor struct {
	queries    *sqlcgen.Queries
	classifier *classify.Classifier
}

// New returns a Processor reading through queries and classifying with
// classifier, which may be nil.
func New(queries *sqlcgen.Queries, classifier *classify.Classifier) *Processor {
	return &Processor{queries: queries, classifier: classifier}
}

// otherExpenseSlug and otherIncomeSlug are the default categories a
// processed email's transaction falls back to when neither a category_hints
// row nor (a later slice's job) an AI call decides it. Matched by slug,
// never by display name: a default's name is resolved through internal/i18n
// and can change independently of this column, the same reason every other
// default-category lookup in this app goes through its slug.
const (
	otherExpenseSlug = "other"
	otherIncomeSlug  = "other_income"
)

// processPendingEmails walks every 'pending' bank_emails row belonging to
// userID and tries to turn each into a transaction. It processes the user's
// whole backlog rather than only the email that just triggered it, which is
// what lets a Render restart that left one mid-flight get swept up by
// whichever email is processed next -- no cron job needed to catch it.
//
// Called as a goroutine kicked off by inboxWebhookHandler after it has
// already written its response, so this always runs against
// context.Background(): the request's own context is canceled the instant
// the handler returns, and using it here would kill this work mid-flight the
// moment the Worker's connection closed. See the same pattern and reasoning
// in auth_password_reset.go's forgot-password email send.
func (p *Processor) ProcessPending(userID int64) {
	ctx := context.Background()
	ids, err := p.queries.ListPendingBankEmailIDs(ctx, userID)
	if err != nil {
		log.Printf("process pending emails: list pending for user %d: %v", userID, err)
		return
	}
	for _, id := range ids {
		p.processOne(ctx, id)
	}
}

// processOneBankEmail claims a single email and either turns it into a
// transaction or closes it as 'ignored'/'failed'. The claim is a single
// UPDATE ... WHERE status='pending' RETURNING (ClaimPendingBankEmail): no
// row back means some other goroutine already has this email, and this call
// must walk away rather than read-then-write, which would let two goroutines
// both process the same message and create two transactions from one email.
func (p *Processor) processOne(ctx context.Context, id int64) {
	email, err := p.queries.ClaimPendingBankEmail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("process bank email %d: claim: %v", id, err)
		return
	}

	notice, err := bankmail.Parse(email.FromAddress, email.Subject, email.Body)
	if err != nil {
		p.markUnprocessable(ctx, email.ID, err)
		return
	}

	// Learn which accounts belong to this person, then decide whether this
	// notice merely moved money between two of them. Both steps run before
	// the transaction is created and neither may prevent it on an error:
	// see rememberOwnAccount and isInternalTransfer.
	p.rememberOwnAccount(ctx, email.UserID, notice.DebitAccount)
	if p.isInternalTransfer(ctx, email.UserID, notice) {
		if markErr := p.queries.MarkBankEmailIgnored(ctx, sqlcgen.MarkBankEmailIgnoredParams{
			ID: email.ID, FailureReason: internalTransferReason,
		}); markErr != nil {
			log.Printf("process bank email %d: mark ignored: %v", id, markErr)
		}
		return
	}

	if err := p.createTransactionFromNotice(ctx, email, notice); err != nil {
		log.Printf("process bank email %d: %v", id, err)
		if markErr := p.queries.MarkBankEmailFailed(ctx, sqlcgen.MarkBankEmailFailedParams{
			ID: email.ID, FailureReason: err.Error(),
		}); markErr != nil {
			log.Printf("process bank email %d: mark failed: %v", id, markErr)
		}
	}
}

// markUnprocessableBankEmail maps a bankmail.Parse error onto the email's
// final status. ErrUnknownSender and ErrNotANotice both mean 'ignored': an
// unrecognized sender or an OTP/ad from a known bank is ordinary mail, not
// something anyone needs to fix. Any other error means 'failed' -- our own
// bug, and the list a person actually needs to look at. Mixing the two would
// flood that failed list with routine mail until nobody reads it anymore.
func (p *Processor) markUnprocessable(ctx context.Context, id int64, err error) {
	if errors.Is(err, bankmail.ErrUnknownSender) || errors.Is(err, bankmail.ErrNotANotice) {
		if markErr := p.queries.MarkBankEmailIgnored(ctx, sqlcgen.MarkBankEmailIgnoredParams{
			ID: id, FailureReason: err.Error(),
		}); markErr != nil {
			log.Printf("process bank email %d: mark ignored: %v", id, markErr)
		}
		return
	}
	if markErr := p.queries.MarkBankEmailFailed(ctx, sqlcgen.MarkBankEmailFailedParams{
		ID: id, FailureReason: err.Error(),
	}); markErr != nil {
		log.Printf("process bank email %d: mark failed: %v", id, markErr)
	}
}

// internalTransferReason is the ignore reason a self-transfer is closed
// with. It is spelled out rather than reusing a bankmail sentinel because
// nothing is wrong with the email: it parsed perfectly, and the reason a
// person reads in the settings list has to say that.
const internalTransferReason = "internal transfer between your own accounts"

// rememberOwnAccount records the debited account as one this person owns.
//
// The debit side of an MB notice is proof of ownership, not a guess: MB
// sends a debit notice to the account holder whose money moved. The
// beneficiary side is deliberately never recorded -- money can be sent to
// anyone, so filing a payee as "yours" would make the next real payment to
// them silently vanish from the ledger.
//
// A write failure is logged and swallowed. Failing to learn an account
// costs at most one self-transfer recorded as an expense, which the owner
// can see and delete; letting it abort the email would cost the whole
// transaction.
func (p *Processor) rememberOwnAccount(ctx context.Context, userID int64, account string) {
	if account == "" {
		return
	}
	if err := p.queries.RememberBankAccount(ctx, sqlcgen.RememberBankAccountParams{
		UserID: userID, AccountNumber: account,
	}); err != nil {
		log.Printf("remember bank account for user %d: %v", userID, err)
	}
}

// isInternalTransfer reports whether this notice only moved money between
// two accounts the same person owns -- neither income nor expense in a
// ledger that treats all of someone's accounts as one pot.
//
// It answers true only on proof: the beneficiary account has itself been
// seen as a debit account on a notice delivered to this person's own inbox.
// A matching account-holder *name* is deliberately not enough. Two people
// can share a name, and the cost of the two mistakes is not symmetric --
// wrongly recording a self-transfer leaves a visible row the owner can
// delete, while wrongly ignoring a real payment removes money from the
// ledger that the owner has no way of knowing is missing.
//
// Every failure answers false, for the same reason: a lookup error must
// cost a spurious row at worst, never a missing one.
func (p *Processor) isInternalTransfer(ctx context.Context, userID int64, notice bankmail.Notice) bool {
	if notice.BeneficiaryAccount == "" || notice.BeneficiaryAccount == notice.DebitAccount {
		return false
	}
	owns, err := p.queries.BankAccountBelongsToUser(ctx, sqlcgen.BankAccountBelongsToUserParams{
		UserID: userID, AccountNumber: notice.BeneficiaryAccount,
	})
	if err != nil {
		log.Printf("check own bank account for user %d: %v", userID, err)
		return false
	}
	return owns
}

// createTransactionFromNotice inserts the transaction a parsed notice
// describes and closes the email as 'imported'. Both steps have to succeed
// for the email to count as done, so a failure here is reported to the
// caller, which marks the email 'failed' instead -- leaving it in
// 'processing' forever would mean it never gets picked up by
// ListPendingBankEmailIDs again.
func (p *Processor) createTransactionFromNotice(ctx context.Context, email sqlcgen.BankEmail, notice bankmail.Notice) error {
	// Computed once and threaded through to both the hint lookup and the
	// insert below, rather than each recomputing it from notice.Description:
	// resolveCategoryForNotice's NoteKey and the row's own stored
	// description must come from the exact same string, or a note long
	// enough to get truncated here would write a hint under a key a later
	// correction (keyed on the already-truncated, stored description) could
	// never reproduce.
	description := truncateEmailNote(notice.Description)

	category, err := p.resolveCategoryForNotice(ctx, email.UserID, notice, description)
	if err != nil {
		return err
	}

	_, err = p.queries.CreateTransactionFromEmail(ctx, sqlcgen.CreateTransactionFromEmailParams{
		UserID:      email.UserID,
		CategoryID:  category.ID,
		Amount:      notice.Amount,
		Type:        notice.Direction,
		Description: description,
		OccurredOn:  pgval.Date(notice.OccurredAt),
		BankEmailID: pgval.Int64(email.ID),
	})
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	if err := p.queries.MarkBankEmailImported(ctx, sqlcgen.MarkBankEmailImportedParams{
		ID:         email.ID,
		OccurredAt: pgtype.Timestamptz{Time: notice.OccurredAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("mark imported: %w", err)
	}
	return nil
}

// resolveCategoryForNotice decides which category a new email-sourced
// transaction files under. A remembered hint for this note wins outright;
// a miss now asks classify.Classifier before falling back to Other/Other
// income, and a successful classification is written back to
// category_hints so the same note key never has to ask again -- that
// write-back is the entire payoff of slice 4's design (see
// classifyAndRememberCategory).
//
// storedDescription must be the same string the caller is about to save on
// the transaction row (truncateEmailNote(notice.Description), not
// notice.Description itself) -- the correction path in txn_mutate.go
// computes its own NoteKey from the row's stored description, and the two
// sides have to agree on exactly the same input or a note long enough to
// get truncated would write a hint under a key that side can never
// reproduce.
//
// A hint is an optimisation, never a requirement for the row to exist at
// all: any error reading category_hints -- not just a plain miss -- falls
// through to the same Other/Other income fallback rather than aborting the
// email. A dropped connection here must cost at most a wrong category, the
// same two-click fix as any other misclassification; it must never cost the
// transaction itself the way returning the error up to processOneBankEmail
// (which marks the email 'failed' and creates nothing) would.
func (p *Processor) resolveCategoryForNotice(ctx context.Context, userID int64, notice bankmail.Notice, storedDescription string) (sqlcgen.Category, error) {
	noteKey := bankmail.NoteKey(storedDescription)
	hint, err := p.queries.GetCategoryHint(ctx, sqlcgen.GetCategoryHintParams{UserID: userID, NoteKey: noteKey})
	switch {
	case err == nil:
		// GetCategoryForUser's own WHERE (user_id = $2 OR user_id IS NULL)
		// is the ownership guard: a hint can only ever have been written for
		// a category this same user owns or a shared default, but this
		// keeps that true even if that ever stopped holding. The type check
		// alongside it guards the other way a hint could misfire -- an old
		// hint whose category was repurposed, or (data corruption aside)
		// one pointing at income when this notice is an expense. Either
		// guard failing falls through to the fallback below rather than
		// erroring: a hint that doesn't fit is exactly a miss, not a bug.
		category, err := p.queries.GetCategoryForUser(ctx, sqlcgen.GetCategoryForUserParams{
			ID: hint.CategoryID, UserID: pgval.Int64(userID),
		})
		if err == nil && category.Type == notice.Direction {
			return category, nil
		}
	case errors.Is(err, pgx.ErrNoRows):
		// A plain miss -- fall through to the fallback below.
	default:
		// Any other lookup error (a dropped connection, an exhausted pool)
		// is logged and treated exactly like a miss -- see the doc comment
		// above for why this must never propagate as an error that costs
		// the transaction.
		log.Printf("look up category hint for user %d: %v", userID, err)
	}

	// No hint fit. Ask Gemini before falling back to Other/Other income --
	// but only when it has something to say: an unconfigured classifier is
	// the ordinary state for an account that never set GEMINI_API_KEY,
	// not a failure worth logging on every single miss.
	if category, err := p.classifyAndRememberCategory(ctx, userID, noteKey, storedDescription, notice.Direction); err == nil {
		return category, nil
	} else if !errors.Is(err, classify.ErrNotConfigured) {
		log.Printf("classify category for user %d: %v", userID, err)
	}

	slug := otherExpenseSlug
	if notice.Direction == "income" {
		slug = otherIncomeSlug
	}
	category, err := p.queries.GetCategoryBySlug(ctx, pgval.Text(slug))
	if err != nil {
		return sqlcgen.Category{}, fmt.Errorf("look up category %q: %w", slug, err)
	}
	return category, nil
}

// classifyAndRememberCategory asks p.classifier which of the user's own
// categories -- filtered to notice's own direction, since an expense
// notice must never be filed under an income category or vice versa --
// fits this note, and writes a successful answer back to category_hints
// under noteKey so the same note never has to ask again. That write-back
// (UpsertCategoryHint) is the entire point of doing this in slice 4 rather
// than calling classify.Classifier on every email forever.
//
// Every failure returns a plain error: an unconfigured or nil Classifier,
// no categories of the right type to offer, the API call itself failing,
// or an answer that names an id outside the candidates just offered. The
// caller (resolveCategoryForNotice) treats all of them identically -- fall
// back to Other/Other income -- which is what makes it safe for this
// function to fail in any of these ways without costing the transaction
// that is about to be created either way.
func (p *Processor) classifyAndRememberCategory(ctx context.Context, userID int64, noteKey, description, direction string) (sqlcgen.Category, error) {
	if p.classifier == nil || !p.classifier.Configured() {
		return sqlcgen.Category{}, classify.ErrNotConfigured
	}

	rows, err := p.queries.ListCategoriesForUser(ctx, pgval.Int64(userID))
	if err != nil {
		return sqlcgen.Category{}, fmt.Errorf("list categories for user %d: %w", userID, err)
	}

	// candidates is keyed by id so the answer can be resolved back to a
	// full row without a second query -- and, as defense in depth, so an
	// id Classify already validated against this same list still can't
	// reach a category outside it if that validation ever had a bug.
	candidates := make(map[int64]sqlcgen.Category, len(rows))
	var options []classify.Category
	for _, row := range rows {
		if row.Type != direction {
			continue
		}
		candidates[row.ID] = row
		options = append(options, classify.Category{ID: row.ID, Name: i18n.CategoryName(row.Slug, row.Name)})
	}
	if len(options) == 0 {
		return sqlcgen.Category{}, fmt.Errorf("classify: no %s categories to offer for user %d", direction, userID)
	}

	categoryID, err := p.classifier.Classify(ctx, description, options)
	if err != nil {
		return sqlcgen.Category{}, err
	}
	category, ok := candidates[categoryID]
	if !ok {
		return sqlcgen.Category{}, fmt.Errorf("classify: model answered category %d, which was not offered to user %d", categoryID, userID)
	}

	if _, err := p.queries.UpsertCategoryHint(ctx, sqlcgen.UpsertCategoryHintParams{
		UserID: userID, NoteKey: noteKey, CategoryID: categoryID,
	}); err != nil {
		// The classification itself succeeded; losing the hint costs a
		// repeat API call the next time this note key shows up, not this
		// transaction's category. Log and keep the answer rather than
		// discarding a good classification over a write that can be
		// retried by simply asking again.
		log.Printf("write category hint for user %d: %v", userID, err)
	}
	return category, nil
}

// truncateEmailNote cuts a note to the limit every other way into the
// ledger enforces, counting runes rather than bytes so a multi-byte
// Vietnamese character is never split mid-character. A note arriving by
// email is trimmed rather than refused: the alternative is losing a real
// transaction over the length of its description.
func truncateEmailNote(s string) string {
	runes := []rune(s)
	if len(runes) <= txnrule.MaxNoteRunes {
		return s
	}
	return string(runes[:txnrule.MaxNoteRunes])
}

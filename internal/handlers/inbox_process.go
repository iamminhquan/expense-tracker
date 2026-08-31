package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"expensetracker/internal/bankmail"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// otherExpenseSlug and otherIncomeSlug are the default categories a
// processed email's transaction falls back to -- learning a better category
// from a user's past corrections is a later slice's job. Matched by slug,
// never by display name: a default's name is resolved through internal/i18n
// and can change independently of this column, the same reason every other
// default-category lookup in this app goes through its slug.
const (
	otherExpenseSlug = "other"
	otherIncomeSlug  = "other_income"
)

// emailNoteMaxRunes mirrors handleCreateTransaction's own 200-character
// limit on the note. Row validation here deliberately repeats what the form
// enforces -- a laxer path in through email would let the processor create
// rows a person typing the same note would have been refused.
const emailNoteMaxRunes = 200

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
func processPendingEmails(deps Deps, userID int64) {
	ctx := context.Background()
	ids, err := deps.Queries.ListPendingBankEmailIDs(ctx, userID)
	if err != nil {
		log.Printf("process pending emails: list pending for user %d: %v", userID, err)
		return
	}
	for _, id := range ids {
		processOneBankEmail(ctx, deps, id)
	}
}

// processOneBankEmail claims a single email and either turns it into a
// transaction or closes it as 'ignored'/'failed'. The claim is a single
// UPDATE ... WHERE status='pending' RETURNING (ClaimPendingBankEmail): no
// row back means some other goroutine already has this email, and this call
// must walk away rather than read-then-write, which would let two goroutines
// both process the same message and create two transactions from one email.
func processOneBankEmail(ctx context.Context, deps Deps, id int64) {
	email, err := deps.Queries.ClaimPendingBankEmail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("process bank email %d: claim: %v", id, err)
		return
	}

	notice, err := bankmail.Parse(email.FromAddress, email.Subject, email.Body)
	if err != nil {
		markUnprocessableBankEmail(ctx, deps, email.ID, err)
		return
	}

	if err := createTransactionFromNotice(ctx, deps, email, notice); err != nil {
		log.Printf("process bank email %d: %v", id, err)
		if markErr := deps.Queries.MarkBankEmailFailed(ctx, sqlcgen.MarkBankEmailFailedParams{
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
func markUnprocessableBankEmail(ctx context.Context, deps Deps, id int64, err error) {
	if errors.Is(err, bankmail.ErrUnknownSender) || errors.Is(err, bankmail.ErrNotANotice) {
		if markErr := deps.Queries.MarkBankEmailIgnored(ctx, sqlcgen.MarkBankEmailIgnoredParams{
			ID: id, FailureReason: err.Error(),
		}); markErr != nil {
			log.Printf("process bank email %d: mark ignored: %v", id, markErr)
		}
		return
	}
	if markErr := deps.Queries.MarkBankEmailFailed(ctx, sqlcgen.MarkBankEmailFailedParams{
		ID: id, FailureReason: err.Error(),
	}); markErr != nil {
		log.Printf("process bank email %d: mark failed: %v", id, markErr)
	}
}

// createTransactionFromNotice inserts the transaction a parsed notice
// describes and closes the email as 'imported'. Both steps have to succeed
// for the email to count as done, so a failure here is reported to the
// caller, which marks the email 'failed' instead -- leaving it in
// 'processing' forever would mean it never gets picked up by
// ListPendingBankEmailIDs again.
func createTransactionFromNotice(ctx context.Context, deps Deps, email sqlcgen.BankEmail, notice bankmail.Notice) error {
	slug := otherExpenseSlug
	if notice.Direction == "income" {
		slug = otherIncomeSlug
	}
	category, err := deps.Queries.GetCategoryBySlug(ctx, pgText(slug))
	if err != nil {
		return fmt.Errorf("look up category %q: %w", slug, err)
	}

	_, err = deps.Queries.CreateTransactionFromEmail(ctx, sqlcgen.CreateTransactionFromEmailParams{
		UserID:      email.UserID,
		CategoryID:  category.ID,
		Amount:      notice.Amount,
		Type:        notice.Direction,
		Description: truncateEmailNote(notice.Description),
		OccurredOn:  pgDate(notice.OccurredAt),
		BankEmailID: pgInt64(email.ID),
	})
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	if err := deps.Queries.MarkBankEmailImported(ctx, sqlcgen.MarkBankEmailImportedParams{
		ID:         email.ID,
		OccurredAt: pgtype.Timestamptz{Time: notice.OccurredAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("mark imported: %w", err)
	}
	return nil
}

// truncateEmailNote cuts a note to the same 200-rune limit
// handleCreateTransaction enforces, counting runes rather than bytes so a
// multi-byte Vietnamese character is never split mid-character.
func truncateEmailNote(s string) string {
	runes := []rune(s)
	if len(runes) <= emailNoteMaxRunes {
		return s
	}
	return string(runes[:emailNoteMaxRunes])
}

package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/bankmail"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// emailSourcedTransaction inserts a transaction the way the processing loop
// would, i.e. with source='email', so a test can exercise the "the owner
// corrects an auto-filed row" path without going through a real bank email.
// bankEmailID is left NULL (the zero pgtype.Int8) since none of these tests
// need a real bank_emails row to hang the transaction off of -- the column
// is nullable for exactly this, ON DELETE SET NULL rather than a hard FK
// requirement.
func emailSourcedTransaction(t *testing.T, deps handlers.Deps, userID, categoryID int64, description string) sqlcgen.Transaction {
	t.Helper()
	txn, err := deps.Queries.CreateTransactionFromEmail(context.Background(), sqlcgen.CreateTransactionFromEmailParams{
		UserID:      userID,
		CategoryID:  categoryID,
		Amount:      15000,
		Type:        "expense",
		Description: description,
		OccurredOn:  pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateTransactionFromEmail() error = %v", err)
	}
	return txn
}

// patchTransactionCategory drives the same PATCH /transactions/{id} route
// the edit row's form submits, changing only the category so the other
// required fields just carry the row's existing values forward.
func patchTransactionCategory(t *testing.T, router http.Handler, cookie *http.Cookie, tok *http.Cookie, txn sqlcgen.Transaction, categoryID int64) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"amount":      {strconv.FormatInt(txn.Amount, 10)},
		"description": {txn.Description},
		"occurred_on": {txn.OccurredOn.Time.Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transactions/%d", txn.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// categoryHintFor reads the raw category_hints row for (userID, noteKey), if
// any, sidestepping the need for a query no production handler uses.
func categoryHintFor(t *testing.T, deps handlers.Deps, userID int64, noteKey string) (categoryID int64, found bool) {
	t.Helper()
	err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM category_hints WHERE user_id = $1 AND note_key = $2", userID, noteKey,
	).Scan(&categoryID)
	if err != nil {
		return 0, false
	}
	return categoryID, true
}

// TestUpdateTransactionOnEmailRowWritesCategoryHint is the correction half
// of the learning loop: editing the category of a source='email' row must
// write (not just update in memory) a category_hints row keyed on the same
// bankmail.NoteKey the processing loop would compute from that row's own
// stored description -- otherwise a later email with the same note would
// never find what this correction just taught the app.
func TestUpdateTransactionOnEmailRowWritesCategoryHint(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-hint-write@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-hint-write@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	fallback := personalCategory(t, deps, user.ID, "Hint Write Fallback")
	corrected := personalCategory(t, deps, user.ID, "Hint Write Corrected")
	const description = "GRAB 8829471 chuyen tien"
	txn := emailSourcedTransaction(t, deps, user.ID, fallback.ID, description)
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	rec := patchTransactionCategory(t, router, cookie, tok, txn, corrected.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /transactions/%d = %d, want 200: %s", txn.ID, rec.Code, rec.Body.String())
	}

	wantKey := bankmail.NoteKey(description)
	gotCategoryID, found := categoryHintFor(t, deps, user.ID, wantKey)
	if !found {
		t.Fatalf("no category_hints row for (user %d, note_key %q) after correcting an email row's category", user.ID, wantKey)
	}
	if gotCategoryID != corrected.ID {
		t.Errorf("category_hints.category_id = %d, want %d (the corrected category)", gotCategoryID, corrected.ID)
	}
}

// TestUpdateTransactionOnManualRowWritesNoCategoryHint guards the other
// half of "the owner corrects a category" -- a plain hand-entered row is
// not something the processing loop will ever see the note of again in the
// same shape, so editing its category must not plant a hint at all.
func TestUpdateTransactionOnManualRowWritesNoCategoryHint(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-hint-manual@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-hint-manual@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	from := personalCategory(t, deps, user.ID, "Manual Hint From")
	to := personalCategory(t, deps, user.ID, "Manual Hint To")
	const description = "manual coffee run"
	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: from.ID, Amount: 15000, Type: "expense",
		Description: description, OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	rec := patchTransactionCategory(t, router, cookie, tok, txn, to.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /transactions/%d = %d, want 200: %s", txn.ID, rec.Code, rec.Body.String())
	}

	if _, found := categoryHintFor(t, deps, user.ID, bankmail.NoteKey(description)); found {
		t.Errorf("category_hints row exists after correcting a manual row's category, want none")
	}
}

// TestUpdateTransactionSecondCorrectionOverwritesCategoryHint is the
// "user changes their mind" case: correcting the same note a second time
// must replace the earlier hint, never add a second row for the same
// (user_id, note_key) -- the ON CONFLICT ... DO UPDATE in
// UpsertCategoryHint is what the unique index on (user_id, note_key) forces
// this to be, and this asserts the handler actually calls it on every
// correction rather than only the first.
func TestUpdateTransactionSecondCorrectionOverwritesCategoryHint(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-hint-overwrite@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-hint-overwrite@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	first := personalCategory(t, deps, user.ID, "Overwrite Hint First")
	second := personalCategory(t, deps, user.ID, "Overwrite Hint Second")
	fallback := personalCategory(t, deps, user.ID, "Overwrite Hint Fallback")
	const description = "NGUYEN VAN A chuyen tien FT24123456789"
	txn := emailSourcedTransaction(t, deps, user.ID, fallback.ID, description)
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	if rec := patchTransactionCategory(t, router, cookie, tok, txn, first.ID); rec.Code != http.StatusOK {
		t.Fatalf("first correction PATCH = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	// Re-fetch: the first PATCH moved the row's own category_id, and the
	// second correction has to start from what the row now holds, the same
	// way a person re-opening the edit row would.
	txn, err = deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("get transaction after first correction: %v", err)
	}
	if rec := patchTransactionCategory(t, router, cookie, tok, txn, second.ID); rec.Code != http.StatusOK {
		t.Fatalf("second correction PATCH = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	noteKey := bankmail.NoteKey(description)
	gotCategoryID, found := categoryHintFor(t, deps, user.ID, noteKey)
	if !found {
		t.Fatalf("no category_hints row for (user %d, note_key %q) after two corrections", user.ID, noteKey)
	}
	if gotCategoryID != second.ID {
		t.Errorf("category_hints.category_id after a second correction = %d, want %d (the latest one)", gotCategoryID, second.ID)
	}
	if n := countRows(t, deps, "SELECT count(*) FROM category_hints WHERE user_id = $1 AND note_key = $2", user.ID, noteKey); n != 1 {
		t.Errorf("category_hints rows for (user %d, note_key %q) = %d, want 1 (overwritten, not duplicated)", user.ID, noteKey, n)
	}
}

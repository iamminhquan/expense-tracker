package handlers

import (
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// txnRow is one row of the list as the template sees it: the query's own row
// plus the things that depend on the view rather than on the transaction --
// whether the date has to name its year, and whether another row on this
// same page looks like the same transaction. transactions.html hands each
// row to the row template on its own, so a page-level flag would not be
// visible from inside it; embedding leaves every existing field reference
// working and adds the ones that were missing.
type txnRow struct {
	sqlcgen.ListTransactionsForMonthRow
	showYear  bool
	duplicate bool
}

// Date is what the row prints in its date column.
func (r txnRow) Date() string { return rowDate(r.OccurredOn, r.showYear) }

// IsDuplicate reports whether another row on this same rendered page shares
// this row's date, amount and type -- see markDuplicates for why it is set
// once while the page's rows are built rather than computed here from a
// query. The single-row fragments handleCreateTransaction, updateTransaction
// and viewTransactionRowHandler build (a create, an edit, a cancelled edit)
// have no sibling rows to compare against, so their "transaction_row" data
// never carries this at all; the template's missing-key lookup then reads as
// false, which is the right answer for a row rendered in isolation.
func (r txnRow) IsDuplicate() bool { return r.duplicate }

// markDuplicates flags every row that shares its date, amount and type with
// another row in the same slice -- the case this exists for is someone
// typing a transaction by hand and the bank email creating it again. It only
// ever compares the rows it is handed, which are deliberately just the ones
// already loaded for one rendered page: reaching further would mean a second
// query for a hint the owner still has to judge for themselves, so two
// duplicates split across different pages go unmarked. That is the accepted
// trade, not a bug -- see Task 5 of the bank-email-slice-2 plan.
func markDuplicates(rows []txnRow) {
	type key struct {
		date   string
		amount int64
		typ    string
	}
	byKey := make(map[key][]int, len(rows))
	for i, r := range rows {
		k := key{date: r.OccurredOn.Time.Format("2006-01-02"), amount: r.Amount, typ: r.Type}
		byKey[k] = append(byKey[k], i)
	}
	for _, idxs := range byKey {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			rows[i].duplicate = true
		}
	}
}

// rowDate is the single rule for that column, kept out of the template
// because which format applies is the view's business, not the markup's.
// The three handlers that answer with one row rather than a list build a
// singleRow instead of a txnRow, and call this directly.
func rowDate(d pgtype.Date, showYear bool) string {
	if showYear {
		return format.DateLong(d)
	}
	return format.DateShort(d)
}

// singleRow is one transaction as the "transaction_row" template sees it
// when it is rendered on its own: the answer to a create, an edit, or a
// cancelled edit, none of which re-render the list around them.
//
// It is a struct rather than the map these three responses each built by
// hand, because the map made a forgotten field invisible: html/template
// prints nothing for a key that is not there, so a response that dropped
// the date shipped an empty column and no error. A missing field here does
// not compile.
type singleRow struct {
	ID            int64
	CategorySlug  pgtype.Text
	CategoryName  string
	CategoryColor string
	Description   string
	Amount        int64
	Type          string
	Source        string
	Date          string
}

// IsDuplicate is always false on a row rendered alone. The badge marks a row
// that shares its date, amount and type with another row on the same
// rendered page, and a single-row response has no siblings to compare
// against -- the list's own rows carry the real answer, see txnRow.
func (r singleRow) IsDuplicate() bool { return false }

// singleRowOf is the row a create or an edit answers with: the transaction
// the write returned, plus the category it was checked against, which the
// same handler has already loaded to validate the write.
func singleRowOf(t sqlcgen.Transaction, c sqlcgen.Category, showYear bool) singleRow {
	return singleRow{
		ID:            t.ID,
		CategorySlug:  c.Slug,
		CategoryName:  c.Name,
		CategoryColor: c.Color,
		Description:   t.Description,
		Amount:        t.Amount,
		Type:          t.Type,
		Source:        t.Source,
		Date:          rowDate(t.OccurredOn, showYear),
	}
}

// editRow is the inline edit form's own state: the values its inputs are
// filled with, the categories its <select> offers, and the message a
// rejected save puts beside them.
type editRow struct {
	ID              int64
	CategoryID      int64
	Description     string
	Amount          int64
	OccurredOnValue string
	CategoryOptions []sqlcgen.Category
	Error           string
}

// dateInputValue formats a DATE column for an <input type="date"> value
// attribute (always "2006-01-02", regardless of display locale) -- distinct
// from dateShort, which is for read-only display.
func dateInputValue(d pgtype.Date) string {
	return d.Time.Format("2006-01-02")
}

func editTransactionRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		txn, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		renderFragment(w, r, deps, "transactions", "transaction_row_edit", editRow{
			ID:              txn.ID,
			CategoryID:      txn.CategoryID,
			Description:     txn.Description,
			Amount:          txn.Amount,
			OccurredOnValue: dateInputValue(txn.OccurredOn),
			CategoryOptions: categoriesOfType(allCategories, txn.Type),
		})
	}
}

func viewTransactionRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		txn, err := deps.Queries.GetTransactionWithCategory(r.Context(), sqlcgen.GetTransactionWithCategoryParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		renderFragment(w, r, deps, "transactions", "transaction_row", singleRow{
			ID:            txn.ID,
			CategorySlug:  txn.CategorySlug,
			CategoryName:  txn.CategoryName,
			CategoryColor: txn.CategoryColor,
			Description:   txn.Description,
			Amount:        txn.Amount,
			Type:          txn.Type,
			Source:        txn.Source,
			Date:          rowDate(txn.OccurredOn, scopeFromRequest(r).All),
		})
	}
}

func deleteConfirmTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}
		if _, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID}); err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}
		renderFragment(w, r, deps, "transactions", "transaction_row_delete_confirm", editRow{ID: id})
	}
}

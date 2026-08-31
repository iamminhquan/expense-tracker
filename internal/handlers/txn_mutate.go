package handlers

import (
	"log"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/bankmail"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
)

// totalsOOBData assembles the payload for the "totals_oob" fragment that
// every create/update/delete returns alongside its row: the refreshed nav
// header balance, plus the transaction count, the month name the list's
// empty state needs, and the pager, whose page count moves whenever a row is
// added or removed.
//
// The header balance is the real current month even when the page is
// browsing an older one, because that is what the widget in the layout
// reports -- which is why it arrives as its own argument rather than being
// derived from anything scoped to the browsed month.
func totalsOOBData(header balanceSummary, count int64, scope txnScope, p pager) map[string]any {
	return map[string]any{
		"HeaderBalance":   header,
		"Count":           count,
		"MonthLabelLower": scope.LabelLower(),
		"AllMonths":       scope.All,
		"Pager":           p,
	}
}

// freshTotals re-reads what every mutation response has to refresh besides
// the row itself: the nav header balance and the count/empty-state for the
// month the originating page is showing.
//
// The two are scoped differently on purpose. The balance widget always
// reports the real current month, because it sits in the layout above the
// month picker; the count belongs to whichever month the page was browsing,
// which is why the window comes from HX-Current-URL rather than from today.
// On failure it writes the 500 itself and reports false.
func freshTotals(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) (map[string]any, bool) {
	scope := scopeFromRequest(r)
	from, to := scope.Bounds()
	header, err := currentHeaderBalance(r, deps, userID)
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return nil, false
	}
	// The count is scoped to the filters the originating page had on as well
	// as to its month: it feeds the chip above the list and the pager below
	// it, both of which describe the rows actually on screen.
	filters := filtersFromHXCurrentURL(r)
	count, err := deps.Queries.CountTransactionsForMonth(r.Context(), filters.countParams(userID, from, to))
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return nil, false
	}
	return totalsOOBData(header, count, scope, newPager(pageFromRequest(r), count, scope.Value)), true
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	form, msg := txnFormFromRequest(r)
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	txnType := r.FormValue("type")
	if txnType != "expense" && txnType != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	source := r.FormValue("ui_source")

	retarget := func(errMsg string) {
		target, fragment := "#quick-add-form-wrapper", "quick_add_form"
		if source == "mobile" {
			target, fragment = "#mobile-quick-add-form", "mobile_quick_add_form"
		}
		w.Header().Set("HX-Retarget", target)
		w.Header().Set("HX-Reswap", "outerHTML")
		renderQuickAddForm(w, r, deps, userID, fragment, errMsg, txnType)
	}

	category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
		ID: form.CategoryID, UserID: pgval.Int64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	if formErr := form.violation(category.Type, txnType); formErr != "" {
		retarget(formErr)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: form.CategoryID, Amount: form.Amount, Type: txnType,
		Description: form.Description, OccurredOn: pgval.Date(form.OccurredOn),
	})
	if err != nil {
		retarget("Could not add the transaction, please try again.")
		return
	}

	w.Header().Set("HX-Trigger", "transaction-created")

	// Three situations where the new row does not belong where the user is
	// looking, and all are answered the same way: re-render the whole month
	// section at page 1 and point htmx at it, instead of the usual single row.
	//
	// Past the first page, a new transaction belongs at the top of page 1, not
	// in whichever window the user happened to be browsing. With a filter on,
	// it may not belong in the list at all -- prepending it would show a row
	// that does not match what the list claims to be showing. And in any order
	// other than the default newest-first, the top is simply not its place:
	// where it goes depends on the amount that was just typed.
	//
	// The month comes back out of the resolved bounds rather than the raw
	// param, so the pushed URL is always a well-formed "YYYY-MM", and it keeps
	// the filters so a reload lands on the same view.
	filters := filtersFromHXCurrentURL(r)
	if pageFromRequest(r) > 1 || filters.Any() || filters.Sorted() {
		month := scopeFromRequest(r).Value
		data, err := buildTransactionsPageData(r, deps, userID, month, 1, filters, "", "")
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}
		// The swap replaces the list section, which the nav sits outside of,
		// so the balance widget still needs its own out-of-band update -- the
		// same one freshTotals folds into the ordinary single-row response.
		header, err := currentHeaderBalance(r, deps, userID)
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}
		data["HeaderBalance"] = header
		w.Header().Set("HX-Retarget", "#transactions-month-section")
		w.Header().Set("HX-Reswap", "outerHTML")
		w.Header().Set("HX-Push-Url", transactionsURL(month, 1, filters))
		renderNamed(w, r, deps, "transactions", "transactions_first_page_response", "transactions", data)
		return
	}

	totals, ok := freshTotals(w, r, deps, userID)
	if !ok {
		return
	}

	renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
		"Row": map[string]any{
			"ID": created.ID, "CategorySlug": category.Slug, "CategoryName": category.Name, "CategoryColor": category.Color,
			"Description": created.Description, "OccurredOn": created.OccurredOn,
			"Amount": created.Amount, "Type": created.Type, "Source": created.Source,
			"Date": rowDate(created.OccurredOn, scopeFromRequest(r).All),
		},
		"Totals": totals,
	})
}

// renderQuickAddForm re-renders one of the two add-transaction forms after a
// validation failure, with the category list filtered to the type the user
// had selected. Which one is the caller's business: handleCreateTransaction
// picks the fragment from ui_source and retargets htmx at the matching
// element, since the desktop form and the mobile sheet are both in the DOM.
func renderQuickAddForm(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, fragment, errMsg, selectedType string) {
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	renderNamed(w, r, deps, "transactions", fragment, "", map[string]any{
		"Categories":    categoriesOfType(allCategories, selectedType),
		"SelectedType":  selectedType,
		"Today":         time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError": errMsg,
	})
}

// categoryPickerHandler answers the hx-get the Expense/Income toggle fires
// when the user flips it: the category control has to be replaced with one
// listing the other type's categories. The desktop form swaps a <select>
// (category_options) and the mobile sheet a row of chips (category_chips) --
// same query, same filter, different markup.
func categoryPickerHandler(deps Deps, fragment string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		typ := r.FormValue("type")
		if typ != "income" {
			typ = "expense"
		}
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		renderNamed(w, r, deps, "transactions", fragment, "", map[string]any{
			"Categories": categoriesOfType(categories, typ),
		})
	}
}

func updateTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		existing, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		form, msg := txnFormFromRequest(r)
		if msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: form.CategoryID, UserID: pgval.Int64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusForbidden)
			return
		}

		// An edit never changes the type: the row keeps the one it was
		// created with, so it is existing.Type the category has to match.
		if formErr := form.violation(category.Type, existing.Type); formErr != "" {
			allCategories, _ := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
			renderNamed(w, r, deps, "transactions", "transaction_row_edit", "", map[string]any{
				"ID": id, "CategoryID": form.CategoryID, "Description": form.Description, "Amount": form.Amount,
				"OccurredOnValue": r.FormValue("occurred_on"),
				"CategoryOptions": categoriesOfType(allCategories, existing.Type), "Error": formErr,
			})
			return
		}

		updated, err := deps.Queries.UpdateTransaction(r.Context(), sqlcgen.UpdateTransactionParams{
			ID: id, UserID: userID, CategoryID: form.CategoryID, Amount: form.Amount, Type: existing.Type,
			Description: form.Description, OccurredOn: pgval.Date(form.OccurredOn),
		})
		if err != nil {
			log.Printf("update transaction: %v", err)
			http.Error(w, "could not update transaction", http.StatusInternalServerError)
			return
		}

		// This is the whole learning mechanism (design doc section 4, step
		// 8): a category correction on an email-sourced row teaches the
		// note-key memory the processing loop consults on the next
		// matching notice, through an action the user already performs --
		// no separate UI. Gated on source == "email" (a manual row's note
		// is never something the processor will key a lookup on the same
		// way) and on the category actually changing (re-saving the same
		// category on every other field edit must not spend a write here).
		// bankmail.NoteKey runs on existing.Description -- the row's
		// description as it stood before this edit, i.e. what the
		// processing loop actually keyed its own lookup on when it created
		// this row -- not the possibly-just-edited description, so a
		// simultaneous description edit can never point the hint at a key
		// the processor never used.
		//
		// existing.Description is already the truncated string
		// inbox_process.go's createTransactionFromNotice stored (its own
		// NoteKey call runs on that same truncated string, never on the raw
		// notice.Description) -- this side must keep deriving its key from
		// whatever is actually stored on the row, not re-truncate or
		// otherwise recompute it, or the two sides drift apart for exactly
		// the longest notes.
		if existing.Source == "email" && form.CategoryID != existing.CategoryID {
			if _, err := deps.Queries.UpsertCategoryHint(r.Context(), sqlcgen.UpsertCategoryHintParams{
				UserID: userID, NoteKey: bankmail.NoteKey(existing.Description), CategoryID: form.CategoryID,
			}); err != nil {
				// The transaction edit itself already succeeded; failing to
				// remember the correction should not turn that into a
				// failed request the user has to retry.
				log.Printf("update transaction: upsert category hint: %v", err)
			}
		}

		totals, ok := freshTotals(w, r, deps, userID)
		if !ok {
			return
		}

		// transaction_create_response despite this being an edit: the
		// fragment is just "a row plus the OOB totals", which is exactly
		// what a successful edit returns too.
		renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
			"Row": map[string]any{
				"ID": updated.ID, "CategorySlug": category.Slug, "CategoryName": category.Name, "CategoryColor": category.Color,
				"Description": updated.Description, "OccurredOn": updated.OccurredOn,
				// Source rides through unchanged from the row UpdateTransaction
				// returns -- editing a category or amount does not turn an
				// email-sourced row into a manual one, so its "auto" label must
				// survive the swap.
				"Amount": updated.Amount, "Type": updated.Type, "Source": updated.Source,
				"Date": rowDate(updated.OccurredOn, scopeFromRequest(r).All),
			},
			"Totals": totals,
		})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{ID: id, UserID: userID}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		totals, ok := freshTotals(w, r, deps, userID)
		if !ok {
			return
		}

		renderNamed(w, r, deps, "transactions", "totals_oob", "", totals)
	}
}

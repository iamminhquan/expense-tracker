package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// categoriesOfType keeps the ones matching typ. The add form, the chips, the
// desktop <select> and the edit row all show categories of one type only,
// and ListCategoriesForUser is the only query that returns them -- so each
// of those paths filters the same list the same way.
func categoriesOfType(categories []sqlcgen.Category, typ string) []sqlcgen.Category {
	var matching []sqlcgen.Category
	for _, c := range categories {
		if c.Type == typ {
			matching = append(matching, c)
		}
	}
	return matching
}

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
// The three handlers that answer with one row rather than a list build a map
// instead of a txnRow, and call this directly.
func rowDate(d pgtype.Date, showYear bool) string {
	if showYear {
		return format.DateLong(d)
	}
	return format.DateShort(d)
}

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

// monthValueOf renders a resolved month bound back into the "YYYY-MM" the
// pager's links carry, so a link built from a month that arrived as an empty
// param still names the month it belongs to instead of dropping back to
// today's on the next click.
func monthValueOf(from pgtype.Date) string { return from.Time.Format("2006-01") }

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateTransaction(w, r, deps, userID)
			return
		}

		query := r.URL.Query()
		filters := filtersFromQuery(query)
		data, err := buildTransactionsPageData(r, deps, userID, query.Get("month"), pageParam(query.Get("page")), filters, "", "")
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		if isFragmentRequest(r) {
			// The filter controls reach the server by serialising their whole
			// form, so the request URL they build carries an empty parameter
			// for every control left blank. Pushing a canonical URL from here
			// instead keeps the address bar to what is actually being
			// filtered -- and makes it the same URL a bookmark or a reload
			// would produce.
			w.Header().Set("HX-Push-Url", transactionsURL(data["MonthValue"].(string), data["Pager"].(pager).Page, filters))
			renderNamed(w, r, deps, "transactions", "transactions_month_section", "transactions", data)
			return
		}
		render(w, r, deps, "transactions", "transactions", data)
	}
}

// buildTransactionsPageData loads everything transactions.html (both the
// full page and the transactions_month_section fragment the month dropdown
// swaps in) needs: the selected month's transactions/totals, the dropdown's
// list of other months with data, and the quick-add form's own state.
func buildTransactionsPageData(r *http.Request, deps Deps, userID int64, monthParam string, page int, filters txnFilters, quickAddError, selectedType string) (map[string]any, error) {
	scope := newTxnScope(monthParam)
	from, to := scope.Bounds()

	// The count comes first: which page exists at all depends on it, and the
	// chip above the list reports every row the filters matched rather than
	// just the page's worth.
	count, err := deps.Queries.CountTransactionsForMonth(r.Context(), filters.countParams(userID, from, to))
	if err != nil {
		return nil, err
	}
	pgr := newPager(page, count, scope.Value)

	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), filters.listParams(userID, from, to, pgr.Offset()))
	if err != nil {
		return nil, err
	}
	rows := make([]txnRow, len(transactions))
	for i, t := range transactions {
		rows[i] = txnRow{ListTransactionsForMonthRow: t, showYear: scope.All}
	}
	markDuplicates(rows)

	months, err := deps.Queries.ListDistinctTransactionMonths(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	currentFrom, _ := currentMonthRange()
	var available []map[string]any
	for _, m := range months {
		if m.Time.Year() == currentFrom.Time.Year() && m.Time.Month() == currentFrom.Time.Month() {
			continue // already offered as the pinned "This month" entry
		}
		available = append(available, map[string]any{
			"Value": m.Time.Format("2006-01"),
			"Label": monthLabel(m.Time),
		})
	}

	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		return nil, err
	}
	formType := selectedType
	if formType != "income" {
		formType = "expense"
	}

	return map[string]any{
		"Transactions":      rows,
		"TotalCount":        count,
		"Pager":             pgr,
		"Filters":           filters,
		"FilterCount":       filters.ActiveCount(),
		"Filtering":         filters.Any(),
		"AllCategories":     allCategories,
		"ExportURL":         exportURL(scope.Value, pgr.Page, filters),
		"MonthValue":        scope.Value,
		"MonthLabel":        scope.Label,
		"MonthLabelLower":   scope.LabelLower(),
		"AllMonths":         scope.All,
		"CurrentMonthValue": currentFrom.Time.Format("2006-01"),
		"AvailableMonths":   available,
		"Categories":        categoriesOfType(allCategories, formType),
		"SelectedType":      selectedType,
		"Today":             time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError":     quickAddError,
		// The importer redirects here with a count rather than rendering its
		// own "done" screen, so the confirmation lands on the list the rows
		// actually went into.
		"Imported": importedCount(r),
	}, nil
}

// importedCount reads the marker the importer redirects with. A malformed
// or missing value is 0, which the banner treats as "no import happened" --
// the same leniency every other value read off this page's URL gets.
func importedCount(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("imported"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category", http.StatusBadRequest)
		return
	}
	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
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
		ID: categoryID, UserID: pgInt64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	var formErr string
	switch {
	case category.Type != txnType:
		formErr = "That category does not match the transaction type"
	case len([]rune(r.FormValue("description"))) > 200:
		formErr = "Note must be 200 characters or fewer"
	case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
		formErr = "That date is too far in the future"
	}
	if formErr != "" {
		retarget(formErr)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: categoryID, Amount: amount, Type: txnType,
		Description: r.FormValue("description"), OccurredOn: pgDate(occurredOn),
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
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
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
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		renderNamed(w, r, deps, "transactions", fragment, "", map[string]any{
			"Categories": categoriesOfType(categories, typ),
		})
	}
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

		allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		renderNamed(w, r, deps, "transactions", "transaction_row_edit", "", map[string]any{
			"ID": txn.ID, "CategoryID": txn.CategoryID, "Description": txn.Description,
			"Amount": txn.Amount, "OccurredOnValue": dateInputValue(txn.OccurredOn),
			"CategoryOptions": categoriesOfType(allCategories, txn.Type),
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

		renderNamed(w, r, deps, "transactions", "transaction_row", "", map[string]any{
			"ID": txn.ID, "CategorySlug": txn.CategorySlug, "CategoryName": txn.CategoryName, "CategoryColor": txn.CategoryColor,
			"Description": txn.Description, "OccurredOn": txn.OccurredOn, "Amount": txn.Amount, "Type": txn.Type,
			"Source": txn.Source,
			"Date":   rowDate(txn.OccurredOn, scopeFromRequest(r).All),
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
		renderNamed(w, r, deps, "transactions", "transaction_row_delete_confirm", "", map[string]any{"ID": id})
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

		categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid category", http.StatusBadRequest)
			return
		}
		amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
		if err != nil || amount <= 0 {
			http.Error(w, "invalid amount", http.StatusBadRequest)
			return
		}
		occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
		if err != nil {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
		description := r.FormValue("description")

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: categoryID, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusForbidden)
			return
		}

		var formErr string
		switch {
		case category.Type != existing.Type:
			formErr = "That category does not match the transaction type"
		case len([]rune(description)) > 200:
			formErr = "Note must be 200 characters or fewer"
		case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
			formErr = "That date is too far in the future"
		}
		if formErr != "" {
			allCategories, _ := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
			renderNamed(w, r, deps, "transactions", "transaction_row_edit", "", map[string]any{
				"ID": id, "CategoryID": categoryID, "Description": description, "Amount": amount,
				"OccurredOnValue": r.FormValue("occurred_on"),
				"CategoryOptions": categoriesOfType(allCategories, existing.Type), "Error": formErr,
			})
			return
		}

		updated, err := deps.Queries.UpdateTransaction(r.Context(), sqlcgen.UpdateTransactionParams{
			ID: id, UserID: userID, CategoryID: categoryID, Amount: amount, Type: existing.Type,
			Description: description, OccurredOn: pgDate(occurredOn),
		})
		if err != nil {
			log.Printf("update transaction: %v", err)
			http.Error(w, "could not update transaction", http.StatusInternalServerError)
			return
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

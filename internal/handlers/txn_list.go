package handlers

import (
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
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

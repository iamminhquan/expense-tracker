package handlers

import (
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/pgval"
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
			w.Header().Set("HX-Push-Url", transactionsURL(data.MonthValue, data.Pager.Page, filters))
			renderViewNamed(w, r, deps, "transactions", "transactions_month_section", "transactions", data)
			return
		}
		renderView(w, r, deps, "transactions", "transactions", data)
	}
}

// buildTransactionsPageData loads everything transactions.html (both the
// full page and the transactions_month_section fragment the month dropdown
// swaps in) needs: the selected month's transactions/totals, the dropdown's
// list of other months with data, and the quick-add form's own state.
// transactionsView is the whole transactions page: the month's rows and the
// count they were drawn from, the filter controls' own state, the month
// picker's entries, and the quick-add form, which is part of the page as
// well as an answer to a rejected submit.
type transactionsView struct {
	viewData

	Transactions []txnRow
	TotalCount   int64
	Pager        pager

	Filters     txnFilters
	FilterCount int
	Filtering   bool
	ExportURL   string

	MonthValue        string
	MonthLabel        string
	MonthLabelLower   string
	AllMonths         bool
	CurrentMonthValue string
	AvailableMonths   []monthOption

	// The add form's own state. AllCategories is the full list the filter
	// bar's category select offers; Categories is the same list narrowed to
	// the type the add form currently has selected.
	AllCategories []sqlcgen.Category
	Categories    []sqlcgen.Category
	SelectedType  string
	Today         string
	QuickAddError string

	// Imported is the marker the importer redirects here with, so the
	// confirmation lands on the list the rows actually went into.
	Imported int
}

func buildTransactionsPageData(r *http.Request, deps Deps, userID int64, monthParam string, page int, filters txnFilters, quickAddError, selectedType string) (*transactionsView, error) {
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

	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
	if err != nil {
		return nil, err
	}
	formType := selectedType
	if formType != "income" {
		formType = "expense"
	}

	return &transactionsView{
		Transactions:      rows,
		TotalCount:        count,
		Pager:             pgr,
		Filters:           filters,
		FilterCount:       filters.ActiveCount(),
		Filtering:         filters.Any(),
		AllCategories:     allCategories,
		ExportURL:         exportURL(scope.Value, pgr.Page, filters),
		MonthValue:        scope.Value,
		MonthLabel:        scope.Label,
		MonthLabelLower:   scope.LabelLower(),
		AllMonths:         scope.All,
		CurrentMonthValue: currentFrom.Time.Format("2006-01"),
		AvailableMonths:   monthOptions(months, currentFrom),
		Categories:        categoriesOfType(allCategories, formType),
		SelectedType:      selectedType,
		Today:             time.Now().In(vietnamLocation).Format("2006-01-02"),
		QuickAddError:     quickAddError,
		Imported:          importedCount(r),
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

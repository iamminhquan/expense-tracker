package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"expensetracker/internal/i18n"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// Everything that answers "which transactions is this page showing?", the
// companion to req_month.go's "which month?". The two work the same way on
// purpose: a value object parsed leniently from the URL, a variant that
// reads the originating page's URL out of HX-Current-URL for the mutation
// handlers, and a canonical rendering back into a query string.

// txnFilters is the transactions list's search box and filter panel as the
// handlers see them. Zero means "not filtering" for each of the three
// numbers -- no category has id 0, and a transaction's amount is validated
// as strictly positive on the way in, so neither sentinel can collide with a
// value a user could actually pick.
//
// Sort is the one member that narrows nothing: it rides here rather than in a
// value object of its own because it sits in the same panel and travels the
// same road as the rest -- the pushed URL, the pager's links, the export, and
// the mutation handlers' HX-Current-URL. Everything that counts what is being
// filtered leaves it out; see Any and ActiveCount.
type txnFilters struct {
	Search    string // matched against a transaction's note, case-insensitively
	Type      string // "expense", "income", or "" for both
	Category  int64
	MinAmount int64
	MaxAmount int64
	Sort      string // "amount_desc", "amount_asc", or "" for newest first
}

// sortOrders is every order the list offers, spelled as the query parameter
// spells them. The set is closed: an order that is not in it is dropped on
// the way in, so a hand-edited "?sort=" cannot follow the view around by way
// of the canonical URL and the export link.
var sortOrders = map[string]bool{"amount_desc": true, "amount_asc": true}

// filtersFromQuery reads the filters out of a parsed query string. Anything
// unusable -- a non-numeric amount, a type that is neither of the two, a
// search term of nothing but spaces -- is dropped rather than reported, in
// the same spirit as pageParam turning a malformed "?page=" into a page that
// exists. A stale bookmark or a hand-edited URL should narrow nothing, not
// produce an error page.
func filtersFromQuery(q url.Values) txnFilters {
	f := txnFilters{Search: strings.TrimSpace(q.Get("q"))}
	if t := q.Get("type"); t == "expense" || t == "income" {
		f.Type = t
	}
	f.Category = positiveInt(q.Get("category"))
	f.MinAmount = positiveInt(q.Get("min"))
	f.MaxAmount = positiveInt(q.Get("max"))
	if s := q.Get("sort"); sortOrders[s] {
		f.Sort = s
	}
	return f
}

// positiveInt parses an id or an amount, returning 0 -- the "not filtering"
// sentinel -- for anything that is not a positive whole number.
func positiveInt(raw string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// filtersFromHXCurrentURL reads the filters the page that issued this
// request was showing. The mutation handlers need it because a
// create/update/delete carries no query string of its own -- see
// currentURLQuery. Without it, the count chip and the pager returned
// alongside a mutated row would describe the unfiltered month and disagree
// with the rows actually on screen.
func filtersFromHXCurrentURL(r *http.Request) txnFilters {
	return filtersFromQuery(currentURLQuery(r))
}

// Any reports whether anything is being filtered at all. The list's empty
// state and the create handler both branch on it: "no transactions in
// August" and "nothing matches your filters" are different things to say,
// and a transaction added while a filter is active may not belong in the
// list it was added from.
func (f txnFilters) Any() bool {
	return f.Search != "" || f.Type != "" || f.Category != 0 || f.MinAmount != 0 || f.MaxAmount != 0
}

// Sorted reports whether the list is in an order other than the default
// newest-first. handleCreateTransaction needs it alongside Any: a new row
// belongs at the top of a list in date order, but in one ordered by amount
// its place depends on the amount that was just typed, so prepending it would
// put it somewhere the order says it does not go.
func (f txnFilters) Sorted() bool { return f.Sort != "" }

// ActiveCount is the number the "Filters" button's badge shows. The amount
// range counts once however many of its two ends are filled in, because the
// badge counts controls the user sees and the panel shows the range as a
// single from-to row.
func (f txnFilters) ActiveCount() int {
	n := 0
	for _, on := range []bool{f.Search != "", f.Type != "", f.Category != 0, f.MinAmount != 0 || f.MaxAmount != 0} {
		if on {
			n++
		}
	}
	return n
}

// searchParam renders the search term for the query's ILIKE. sqlc gives a
// nullable text parameter as pgtype.Text; an invalid one is SQL NULL, which
// is what switches the predicate off.
func (f txnFilters) searchParam() pgtype.Text {
	if f.Search == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: f.Search, Valid: true}
}

// searchSlugs is the other half of the search: the default categories whose
// displayed label contains the term. Their labels live in internal/i18n
// rather than in the database, so the term is resolved into slugs here and
// the query matches on those -- a row showing "Transport" is found by
// searching for it, without the SQL having to know what any slug is called.
//
// It always returns a non-nil slice so the parameter reaches Postgres as an
// empty array rather than NULL, which keeps the predicate plain true-or-false
// instead of dragging three-valued logic through the OR chain around it.
func (f txnFilters) searchSlugs() []string {
	slugs := i18n.SlugsMatching(f.Search)
	if slugs == nil {
		return []string{}
	}
	return slugs
}

func (f txnFilters) typeParam() pgtype.Text {
	if f.Type == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: f.Type, Valid: true}
}

// sortParam renders the order for the query's ORDER BY. An unset one is SQL
// NULL, which matches neither CASE and so leaves the list newest-first.
func (f txnFilters) sortParam() pgtype.Text {
	if f.Sort == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: f.Sort, Valid: true}
}

// nullableInt turns a 0 sentinel into SQL NULL and anything else into a set
// parameter, for the three numeric filters.
func nullableInt(v int64) pgtype.Int8 {
	if v == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: v, Valid: true}
}

// transactionsURL renders the canonical address of one view of the list:
// which month, which page, and what is being filtered. Fragment responses
// push it via the HX-Push-Url header rather than letting htmx push the
// request URL it happened to build, so the address bar never accumulates the
// empty "q=&type=&category=" that serialising the whole filter form
// produces. Its output is also what the pager and the month picker are read
// back out of on the next request.
func transactionsURL(month string, page int, f txnFilters) string {
	q := filterQuery(month, f)
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	return "/transactions?" + q.Encode()
}

// exportURL renders the address the Export link points at: the same month
// and the same filters, so the CSV is what is on screen. The page is
// deliberately left out -- the export is not paginated, and a link that
// carried "page=3" would quietly hand back a third of the month.
func exportURL(month string, f txnFilters) string {
	return "/transactions/export?" + filterQuery(month, f).Encode()
}

// filterQuery is the part both addresses agree on: the month, plus only
// those filters that are actually set. Leaving the unset ones out is what
// keeps the address bar free of the "q=&type=&category=" that serialising
// the whole filter form would produce.
func filterQuery(month string, f txnFilters) url.Values {
	q := url.Values{"month": {month}}
	if f.Search != "" {
		q.Set("q", f.Search)
	}
	if f.Type != "" {
		q.Set("type", f.Type)
	}
	for name, v := range map[string]int64{"category": f.Category, "min": f.MinAmount, "max": f.MaxAmount} {
		if v != 0 {
			q.Set(name, strconv.FormatInt(v, 10))
		}
	}
	// The order is not a filter, but it is part of the view: without it here a
	// reload, a page turn or an export would silently drop back to newest-first.
	if f.Sort != "" {
		q.Set("sort", f.Sort)
	}
	return q
}

// exportParams, listParams and countParams build the query parameter sets
// from one filter value, so the five nullable predicates are spelled out
// once rather than at every call site. The queries' WHERE clauses are
// identical by design, and these keep the Go side of that promise.

// exportParams selects every row the filters match, with no page window:
// leaving Limit and Offset unset sends both as SQL NULL, which Postgres
// reads as LIMIT ALL OFFSET 0. The CSV export therefore runs the same query
// as the list it was launched from rather than a second one that could
// drift from it.
func (f txnFilters) exportParams(userID int64, from, to pgtype.Date) sqlcgen.ListTransactionsForMonthParams {
	return sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: f.searchParam(), SearchSlugs: f.searchSlugs(), Type: f.typeParam(),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
		Sort: f.sortParam(),
	}
}

// listParams narrows exportParams to the one page the list is showing.
func (f txnFilters) listParams(userID int64, from, to pgtype.Date, offset int32) sqlcgen.ListTransactionsForMonthParams {
	params := f.exportParams(userID, from, to)
	params.Limit = pgtype.Int4{Int32: pageSize, Valid: true}
	params.Offset = pgtype.Int4{Int32: offset, Valid: true}
	return params
}

func (f txnFilters) countParams(userID int64, from, to pgtype.Date) sqlcgen.CountTransactionsForMonthParams {
	return sqlcgen.CountTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: f.searchParam(), SearchSlugs: f.searchSlugs(), Type: f.typeParam(),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
	}
}

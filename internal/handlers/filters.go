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
// companion to month.go's "which month?". The two work the same way on
// purpose: a value object parsed leniently from the URL, a variant that
// reads the originating page's URL out of HX-Current-URL for the mutation
// handlers, and a canonical rendering back into a query string.

// txnFilters is the transactions list's search box and filter panel as the
// handlers see them. Zero means "not filtering" for each of the three
// numbers -- no category has id 0, and a transaction's amount is validated
// as strictly positive on the way in, so neither sentinel can collide with a
// value a user could actually pick.
type txnFilters struct {
	Search    string // matched against a transaction's note, case-insensitively
	Type      string // "expense", "income", or "" for both
	Category  int64
	MinAmount int64
	MaxAmount int64
}

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

// filtersFromRequest reads the filters the request's own URL carries, for
// the list page itself.
func filtersFromRequest(r *http.Request) txnFilters {
	return filtersFromQuery(r.URL.Query())
}

// filtersFromHXCurrentURL reads the filters the page that issued this
// request was showing. The mutation handlers need this for the same reason
// monthRangeFromRequest exists: a create/update/delete carries no query
// string of its own, but htmx sends the originating page's full URL on every
// request it issues. Without it, the count chip and the pager returned
// alongside a mutated row would describe the unfiltered month and disagree
// with the rows actually on screen.
func filtersFromHXCurrentURL(r *http.Request) txnFilters {
	raw := r.Header.Get("HX-Current-URL")
	if raw == "" {
		return txnFilters{}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return txnFilters{}
	}
	return filtersFromQuery(u.Query())
}

// Any reports whether anything is being filtered at all. The list's empty
// state and the create handler both branch on it: "no transactions in
// August" and "nothing matches your filters" are different things to say,
// and a transaction added while a filter is active may not belong in the
// list it was added from.
func (f txnFilters) Any() bool {
	return f.Search != "" || f.Type != "" || f.Category != 0 || f.MinAmount != 0 || f.MaxAmount != 0
}

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
	q := url.Values{"month": {month}}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
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
	return "/transactions?" + q.Encode()
}

// listParams and countParams build the two query parameter sets from one
// filter value, so the five nullable predicates are spelled out once rather
// than at every call site. The queries' WHERE clauses are identical by
// design, and these keep the Go side of that promise.
func (f txnFilters) listParams(userID int64, from, to pgtype.Date, offset int32) sqlcgen.ListTransactionsForMonthParams {
	return sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: f.searchParam(), SearchSlugs: f.searchSlugs(), Type: f.typeParam(),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
		Limit: pageSize, Offset: offset,
	}
}

func (f txnFilters) countParams(userID int64, from, to pgtype.Date) sqlcgen.CountTransactionsForMonthParams {
	return sqlcgen.CountTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: f.searchParam(), SearchSlugs: f.searchSlugs(), Type: f.typeParam(),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
	}
}

package api

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// This file duplicates handlers/req_month.go, req_filters.go, and
// req_paging.go's value objects -- the part of the HTML side's "which
// transactions is this page showing?" that has nothing to do with HTML.
// Dropped on the way over: HX-Current-URL reading and the canonical
// ?month=&page=&... URL builders, which only existed because an htmx
// mutation POST carries no query string of its own to recover filters
// from. A JSON client has no such problem -- it already holds the filter
// state it sent and resends it on the next request -- so there is no
// "scopeFromRequest"/"filtersFromHXCurrentURL" equivalent here at all.

// --- month scope ---

const allMonthsValue = "all"

var (
	allTimeFrom = pgval.Date(time.Date(1, 1, 1, 0, 0, 0, 0, vietnamLocation))
	allTimeTo   = pgval.Date(time.Date(9999, 12, 31, 0, 0, 0, 0, vietnamLocation))
)

// monthScope is which slice of time a transactions request is scoped to:
// one calendar month, or every month there has ever been. See
// handlers.txnScope's doc comment for the full reasoning; it applies
// unchanged here.
type monthScope struct {
	Value string
	Label string
	All   bool

	from, to pgtype.Date
}

func newMonthScope(param string) monthScope {
	if param == allMonthsValue {
		return monthScope{Value: allMonthsValue, Label: "All months", All: true, from: allTimeFrom, to: allTimeTo}
	}
	from, to := monthRangeFor(param)
	return monthScope{Value: from.Time.Format("2006-01"), Label: monthLabel(from.Time), from: from, to: to}
}

func (s monthScope) Bounds() (from, to pgtype.Date) { return s.from, s.to }
func (s monthScope) LabelLower() string             { return s.from.Time.Format("January") }

func monthRangeFor(param string) (from, to pgtype.Date) {
	t, err := time.ParseInLocation("2006-01", param, vietnamLocation)
	if err != nil {
		return currentMonthRange()
	}
	fromTime := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, vietnamLocation)
	return pgval.Date(fromTime), pgval.Date(fromTime.AddDate(0, 1, 0))
}

func currentMonthRange() (from, to pgtype.Date) {
	now := time.Now().In(vietnamLocation)
	fromTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, vietnamLocation)
	return pgval.Date(fromTime), pgval.Date(fromTime.AddDate(0, 1, 0))
}

func monthLabel(t time.Time) string { return t.Format("January 2006") }

// monthOptionDTO is one entry the month dropdown offers, mirroring
// handlers.monthOption.
type monthOptionDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func monthOptions(months []pgtype.Date, current pgtype.Date) []monthOptionDTO {
	var options []monthOptionDTO
	for _, m := range months {
		if m.Time.Year() == current.Time.Year() && m.Time.Month() == current.Time.Month() {
			continue
		}
		options = append(options, monthOptionDTO{Value: m.Time.Format("2006-01"), Label: monthLabel(m.Time)})
	}
	return options
}

// --- filters ---

// txnFilters mirrors handlers.txnFilters field for field; see its doc
// comment for why Sort travels with the rest despite not narrowing
// anything.
type txnFilters struct {
	Search    string
	Type      string
	Category  int64
	MinAmount int64
	MaxAmount int64
	Sort      string
}

var sortOrders = map[string]bool{"amount_desc": true, "amount_asc": true}

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

func positiveInt(raw string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func (f txnFilters) Any() bool {
	return f.Search != "" || f.Type != "" || f.Category != 0 || f.MinAmount != 0 || f.MaxAmount != 0
}

func (f txnFilters) ActiveCount() int {
	n := 0
	for _, on := range []bool{f.Search != "", f.Type != "", f.Category != 0, f.MinAmount != 0 || f.MaxAmount != 0} {
		if on {
			n++
		}
	}
	return n
}

func (f txnFilters) searchSlugs() []string {
	slugs := i18n.SlugsMatching(f.Search)
	if slugs == nil {
		return []string{}
	}
	return slugs
}

func nullableText(v string) pgtype.Text {
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func nullableInt(v int64) pgtype.Int8 {
	if v == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: v, Valid: true}
}

func (f txnFilters) exportParams(userID int64, from, to pgtype.Date) sqlcgen.ListTransactionsForMonthParams {
	return sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: nullableText(f.Search), SearchSlugs: f.searchSlugs(), Type: nullableText(f.Type),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
		Sort: nullableText(f.Sort),
	}
}

func (f txnFilters) listParams(userID int64, from, to pgtype.Date, offset int32) sqlcgen.ListTransactionsForMonthParams {
	params := f.exportParams(userID, from, to)
	params.Limit = pgtype.Int4{Int32: pageSize, Valid: true}
	params.Offset = pgtype.Int4{Int32: offset, Valid: true}
	return params
}

func (f txnFilters) countParams(userID int64, from, to pgtype.Date) sqlcgen.CountTransactionsForMonthParams {
	return sqlcgen.CountTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
		Search: nullableText(f.Search), SearchSlugs: f.searchSlugs(), Type: nullableText(f.Type),
		CategoryID: nullableInt(f.Category),
		MinAmount:  nullableInt(f.MinAmount), MaxAmount: nullableInt(f.MaxAmount),
	}
}

// --- paging ---

// pageSize mirrors handlers.pageSize. Change it here *and* there until
// Phase 4 deletes one of the two copies.
const pageSize = 10

// pagerDTO is what the client needs to draw pagination controls: nothing
// here recomputes a page number from a URL the way handlers.pager's
// MonthValue field does, because the JSON response always states its own
// month explicitly (see listTransactionsResponse).
type pagerDTO struct {
	Page       int  `json:"page"`
	TotalPages int  `json:"totalPages"`
	HasPrev    bool `json:"hasPrev"`
	HasNext    bool `json:"hasNext"`
}

func newPager(requested int, total int64) pagerDTO {
	totalPages := int((total + pageSize - 1) / pageSize)
	if totalPages < 1 {
		totalPages = 1
	}
	page := requested
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return pagerDTO{Page: page, TotalPages: totalPages, HasPrev: page > 1, HasNext: page < totalPages}
}

func (p pagerDTO) offset() int32 { return int32((p.Page - 1) * pageSize) }

func pageParam(raw string) int {
	page, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return page
}

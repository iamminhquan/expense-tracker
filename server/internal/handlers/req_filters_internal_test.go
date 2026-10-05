package handlers

import (
	"net/url"
	"testing"
)

func TestFiltersFromQueryReadsEveryCriterion(t *testing.T) {
	f := filtersFromQuery(url.Values{
		"q":        {"coffee"},
		"type":     {"expense"},
		"category": {"7"},
		"min":      {"50000"},
		"max":      {"200000"},
	})

	if f.Search != "coffee" {
		t.Errorf("expected search %q, got %q", "coffee", f.Search)
	}
	if f.Type != "expense" {
		t.Errorf("expected type %q, got %q", "expense", f.Type)
	}
	if f.Category != 7 {
		t.Errorf("expected category 7, got %d", f.Category)
	}
	if f.MinAmount != 50000 {
		t.Errorf("expected min 50000, got %d", f.MinAmount)
	}
	if f.MaxAmount != 200000 {
		t.Errorf("expected max 200000, got %d", f.MaxAmount)
	}
}

// A malformed filter is dropped rather than raising an error, in the same
// spirit as pageParam clamping a nonsensical "?page=": a bad value in a URL
// the user typed or a stale bookmark should narrow nothing, not produce an
// error page.
func TestFiltersFromQueryDropsUnusableValues(t *testing.T) {
	f := filtersFromQuery(url.Values{
		"q":        {"   "},
		"type":     {"transfer"},
		"category": {"abc"},
		"min":      {"-5"},
		"max":      {"9,000"},
	})

	if f.Any() {
		t.Errorf("expected every unusable value to be dropped, got %+v", f)
	}
}

func TestFiltersActiveCountCountsOnlyWhatIsSet(t *testing.T) {
	if n := (txnFilters{}).ActiveCount(); n != 0 {
		t.Errorf("expected an empty filter set to count 0, got %d", n)
	}
	f := txnFilters{Search: "coffee", Type: "expense", Category: 7}
	if n := f.ActiveCount(); n != 3 {
		t.Errorf("expected 3 active filters, got %d", n)
	}
}

// The amount range counts as one filter however many of its two ends are
// filled in: the badge counts controls the user sees, and the panel shows
// "Amount" as a single from-to row.
func TestFiltersAmountRangeCountsAsOneFilter(t *testing.T) {
	if n := (txnFilters{MinAmount: 1000}).ActiveCount(); n != 1 {
		t.Errorf("expected a lone minimum to count as 1, got %d", n)
	}
	if n := (txnFilters{MinAmount: 1000, MaxAmount: 9000}).ActiveCount(); n != 1 {
		t.Errorf("expected a full range to still count as 1, got %d", n)
	}
}

// The canonical URL the fragment responses push. It carries only what is
// actually set, so filtering never leaves "?q=&type=&category=" behind in
// the address bar the way form serialization would.
func TestFiltersCanonicalURLCarriesOnlyWhatIsSet(t *testing.T) {
	got := transactionsURL("2026-08", 1, txnFilters{Search: "coffee", Type: "expense"})
	if got != "/transactions?month=2026-08&q=coffee&type=expense" {
		t.Errorf("unexpected canonical URL: %s", got)
	}
}

func TestFiltersCanonicalURLKeepsAPageBeyondTheFirst(t *testing.T) {
	got := transactionsURL("2026-08", 3, txnFilters{})
	if got != "/transactions?month=2026-08&page=3" {
		t.Errorf("unexpected canonical URL: %s", got)
	}
}

func TestFiltersCanonicalURLEscapesTheSearchTerm(t *testing.T) {
	got := transactionsURL("2026-08", 1, txnFilters{Search: "cà phê & bánh"})
	if got != "/transactions?month=2026-08&q=c%C3%A0+ph%C3%AA+%26+b%C3%A1nh" {
		t.Errorf("unexpected canonical URL: %s", got)
	}
}

// The Export link's address. It carries the same month and the same filters
// as the list, so the CSV matches what is on screen -- but never a page,
// since the export is not paginated and "page 2 of the download" is not a
// thing a user asked for.
func TestExportURLCarriesTheMonthAndFiltersButNeverThePage(t *testing.T) {
	got := exportURL("2026-08", txnFilters{Search: "coffee", Type: "expense"})
	if got != "/transactions/export?month=2026-08&q=coffee&type=expense" {
		t.Errorf("unexpected export URL: %s", got)
	}
}

func TestFiltersFromQueryReadsTheSortOrder(t *testing.T) {
	for _, want := range []string{"amount_desc", "amount_asc"} {
		if got := filtersFromQuery(url.Values{"sort": {want}}).Sort; got != want {
			t.Errorf("filtersFromQuery(sort=%q).Sort = %q, want %q", want, got, want)
		}
	}
}

// An unknown order falls back to the default rather than reaching the query,
// for the same reason a malformed amount is dropped: the ORDER BY is built
// from this value, and only the orders the control offers may reach it.
func TestFiltersFromQueryDropsAnUnknownSortOrder(t *testing.T) {
	for _, raw := range []string{"amount", "occurred_on DESC", "", "  "} {
		if got := filtersFromQuery(url.Values{"sort": {raw}}).Sort; got != "" {
			t.Errorf("filtersFromQuery(sort=%q).Sort = %q, want %q", raw, got, "")
		}
	}
}

// Sorting hides no rows, so it is not one of the things the badge counts or
// the "no transactions match these filters" empty state is about.
func TestSortIsNotCountedAsAFilter(t *testing.T) {
	f := txnFilters{Sort: "amount_desc"}
	if f.Any() {
		t.Error("expected a sort order alone to leave the list unfiltered")
	}
	if n := f.ActiveCount(); n != 0 {
		t.Errorf("txnFilters{Sort: \"amount_desc\"}.ActiveCount() = %d, want 0", n)
	}
}

// Sorted is what tells handleCreateTransaction that a new row cannot simply
// be prepended: outside the default date order its place depends on the
// amount that was just typed.
func TestSortedReportsOnlyANonDefaultOrder(t *testing.T) {
	if (txnFilters{}).Sorted() {
		t.Error("expected the default order to report as unsorted")
	}
	if !(txnFilters{Sort: "amount_asc"}).Sorted() {
		t.Error("expected an amount order to report as sorted")
	}
}

// The order is part of the view, so it rides along in every address the view
// is rebuilt from: the pushed URL, the pager's links and the export.
func TestFiltersCanonicalURLCarriesTheSortOrder(t *testing.T) {
	got := transactionsURL("2026-08", 1, txnFilters{Sort: "amount_desc"})
	if got != "/transactions?month=2026-08&sort=amount_desc" {
		t.Errorf("unexpected canonical URL: %s", got)
	}
}

func TestExportURLCarriesTheSortOrder(t *testing.T) {
	got := exportURL("2026-08", txnFilters{Sort: "amount_asc"})
	if got != "/transactions/export?month=2026-08&sort=amount_asc" {
		t.Errorf("unexpected export URL: %s", got)
	}
}

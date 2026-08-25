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

package i18n

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func slug(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func TestCategoryName(t *testing.T) {
	tests := []struct {
		name string
		slug pgtype.Text
		raw  string
		want string
	}{
		{"default category shows its translation", slug("food_drink"), "Food & Drink", "Food & Drink"},
		{"translation wins over a stale name column", slug("transport"), "Đi lại", "Transport"},
		{"user category has no slug and keeps its own name", pgtype.Text{}, "Cà phê", "Cà phê"},
		{"unknown slug falls back to the name column", slug("groceries"), "Groceries", "Groceries"},
		{"empty-but-valid slug falls back too", slug(""), "Cà phê", "Cà phê"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CategoryName(tc.slug, tc.raw); got != tc.want {
				t.Errorf("CategoryName(%q, %q) = %q, want %q", tc.slug.String, tc.raw, got, tc.want)
			}
		})
	}
}

// A category row must never render as a blank label, whatever the database
// holds -- an empty cell in the list gives the user nothing to click on and
// no way to tell which category a transaction belongs to.
func TestCategoryNameNeverEmptyWhenNameIsPresent(t *testing.T) {
	for _, s := range []pgtype.Text{{}, slug(""), slug("nope"), slug("other")} {
		if got := CategoryName(s, "Fallback"); got == "" {
			t.Errorf("CategoryName(%+v, %q) returned empty", s, "Fallback")
		}
	}
}

func TestNameForSlug(t *testing.T) {
	if got := NameForSlug("other"); got != "Other" {
		t.Errorf(`NameForSlug("other") = %q, want "Other"`, got)
	}
	if got := NameForSlug("nope"); got != "" {
		t.Errorf(`NameForSlug("nope") = %q, want ""`, got)
	}
}

// Every slug the 000008 migration writes must be resolvable, or that
// category renders with whatever the migration left in its name column
// instead of its translation.
func TestEverySeededSlugHasATranslation(t *testing.T) {
	seeded := []string{
		"food_drink", "transport", "entertainment", "bills", "health",
		"shopping", "other", "salary", "bonus", "other_income",
	}
	for _, s := range seeded {
		if NameForSlug(s) == "" {
			t.Errorf("slug %q is written by migration 000008 but has no translation", s)
		}
	}
}

// The transactions search box matches category names as well as notes, and
// what it has to match is the name the row displays -- which for a default
// category is the label here, not the name column.
func TestSlugsMatchingFindsDefaultsByTheirDisplayedName(t *testing.T) {
	got := SlugsMatching("transport")
	if len(got) != 1 || got[0] != "transport" {
		t.Errorf(`expected ["transport"], got %v`, got)
	}
}

func TestSlugsMatchingIsCaseInsensitiveAndPartial(t *testing.T) {
	if got := SlugsMatching("DRINK"); len(got) != 1 || got[0] != "food_drink" {
		t.Errorf(`expected ["food_drink"] for a partial, case-folded term, got %v`, got)
	}
}

// "Other" labels two defaults -- one on each side of the ledger -- and a
// search for it should reach both.
func TestSlugsMatchingReturnsEverySlugTheTermLabels(t *testing.T) {
	got := SlugsMatching("other")
	found := map[string]bool{}
	for _, s := range got {
		found[s] = true
	}
	if !found["other"] || !found["other_income"] {
		t.Errorf(`expected both "other" and "other_income", got %v`, got)
	}
}

func TestSlugsMatchingReturnsNothingForATermNoLabelContains(t *testing.T) {
	if got := SlugsMatching("zzz"); len(got) != 0 {
		t.Errorf("expected no slugs, got %v", got)
	}
}

// An empty term is not "matches everything": it means the search box is
// empty, and every label would otherwise contain it.
func TestSlugsMatchingReturnsNothingForAnEmptyTerm(t *testing.T) {
	if got := SlugsMatching("  "); len(got) != 0 {
		t.Errorf("expected an empty term to match no slugs, got %v", got)
	}
}

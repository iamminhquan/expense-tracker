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

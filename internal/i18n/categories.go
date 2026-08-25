// Package i18n resolves the display name of a default category from its
// slug.
//
// Only category names go through here. Every other string in the app is
// written in English directly in the template or handler that shows it --
// there is no message catalog, and adding one is a separate job. Categories
// are the exception because their names live in the database rather than in
// a template, so a future language switcher could not reach them without a
// key that is independent of what is displayed.
package i18n

import (
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// categoryNames maps the slug of each shared default category to its
// English label. Categories a user creates have no slug and are never in
// here -- their names are the user's own words, not ours to translate.
var categoryNames = map[string]string{
	"food_drink":    "Food & Drink",
	"transport":     "Transport",
	"entertainment": "Entertainment",
	"bills":         "Bills",
	"health":        "Health",
	"shopping":      "Shopping",
	"other":         "Other",
	"salary":        "Salary",
	"bonus":         "Bonus",
	"other_income":  "Other income",
}

// NameForSlug returns the label for a default category slug, or "" if the
// slug is unknown. Callers holding a row should prefer CategoryName, which
// has a name to fall back on; this is for the places that synthesise a
// category out of nothing, such as the pie chart's aggregate slice.
func NameForSlug(slug string) string {
	return categoryNames[slug]
}

// CategoryName returns what to show for a category row. A user-created
// category has a NULL slug and shows the name it was given; a default
// category shows its translation. A slug that is not in the map -- a
// default added by a later migration than this build knows about -- falls
// back to the name column too, which is why migrations keep that column
// populated with a usable English label rather than leaving it stale.
func CategoryName(slug pgtype.Text, name string) string {
	if slug.Valid {
		if translated := categoryNames[slug.String]; translated != "" {
			return translated
		}
	}
	return name
}

// SlugsMatching returns the slugs of every default category whose displayed
// label contains term, case-insensitively.
//
// The transactions search box needs this because a default category's label
// lives here rather than in the database: searching the name column would
// match whatever a migration happened to leave there, not what the row on
// screen actually says. Translating the term into slugs keeps the match on
// the one thing that identifies a default category.
//
// An empty term matches nothing rather than everything -- it means the search
// box is empty, and every label trivially contains "".
func SlugsMatching(term string) []string {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}
	folded := strings.ToLower(term)
	var slugs []string
	for slug, label := range categoryNames {
		if strings.Contains(strings.ToLower(label), folded) {
			slugs = append(slugs, slug)
		}
	}
	// Map iteration order is random; sorting keeps the query parameter (and
	// anything that logs it) stable from one request to the next.
	sort.Strings(slugs)
	return slugs
}

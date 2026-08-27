package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/csvimport"
	"expensetracker/internal/i18n"
)

// mappingFields names the form control behind each column role. The names
// are the one place the Go side and the template have to agree, so they are
// written once and read back through the same table.
var mappingFields = []struct {
	Label    string
	Field    string
	Optional bool
	get      func(*csvimport.Mapping) *int
}{
	{"Date", "date_col", false, func(m *csvimport.Mapping) *int { return &m.Date }},
	{"Amount", "amount_col", false, func(m *csvimport.Mapping) *int { return &m.Amount }},
	{"Type", "type_col", true, func(m *csvimport.Mapping) *int { return &m.Type }},
	{"Category", "category_col", true, func(m *csvimport.Mapping) *int { return &m.Category }},
	{"Note", "note_col", true, func(m *csvimport.Mapping) *int { return &m.Note }},
}

// mappingRole is one row of the mapping screen: a label, the control's
// name, and which column is currently chosen.
type mappingRole struct {
	Label    string
	Field    string
	Optional bool
	Selected int
}

// mappingFromForm reads back a mapping the browser is carrying. It reports
// false when the form holds no mapping at all, which is what tells the
// handler this upload has not been through the mapping screen yet.
//
// Nothing here rejects a value: an index that does not exist, or a date
// layout nobody offers, is caught by validateMapping, which can say which
// control is wrong. This is the same lenient-parse-then-validate split the
// month and filter value objects use.
func mappingFromForm(r *http.Request) (csvimport.Mapping, bool) {
	if r.FormValue("mapped") == "" {
		return csvimport.Mapping{}, false
	}
	m := csvimport.Mapping{
		DateLayout:        r.FormValue("date_layout"),
		NegativeIsExpense: r.FormValue("negative_is_expense") != "",
		FallbackCategory:  strings.TrimSpace(r.FormValue("fallback_category")),
	}
	for _, f := range mappingFields {
		*f.get(&m) = columnIndex(r.FormValue(f.Field))
	}
	return m, true
}

// columnIndex reads one column select. Anything unparseable is "no column",
// which for the two required roles becomes a validation error naming them.
func columnIndex(value string) int {
	i, err := strconv.Atoi(value)
	if err != nil || i < 0 {
		return csvimport.NoColumn
	}
	return i
}

// validateMapping returns what is wrong with a submitted mapping, or "".
func validateMapping(m csvimport.Mapping, columns int) string {
	if m.Date == csvimport.NoColumn {
		return "Pick the column that holds the date."
	}
	if m.Amount == csvimport.NoColumn {
		return "Pick the column that holds the amount."
	}
	if !csvimport.ValidDateLayout(m.DateLayout) {
		return "Pick how the dates in this file are written."
	}
	if m.Category == csvimport.NoColumn && m.FallbackCategory == "" {
		return "This file has no category column, so pick a category to file everything under."
	}

	used := map[int]string{}
	for _, f := range mappingFields {
		copied := m
		i := *f.get(&copied)
		if i == csvimport.NoColumn {
			continue
		}
		if i >= columns {
			return "That column is not in this file. Pick again."
		}
		if already, taken := used[i]; taken {
			return "One column cannot be both " + already + " and " + f.Label + "."
		}
		used[i] = f.Label
	}
	return ""
}

// importMapping renders the screen that asks how to read a file the app did
// not write.
func importMapping(w http.ResponseWriter, r *http.Request, deps Deps, sheet *csvimport.Sheet, m csvimport.Mapping, errMsg string) {
	userID, _ := auth.UserIDFromContext(r.Context())

	roles := make([]mappingRole, 0, len(mappingFields))
	for _, f := range mappingFields {
		copied := m
		roles = append(roles, mappingRole{
			Label: f.Label, Field: f.Field, Optional: f.Optional, Selected: *f.get(&copied),
		})
	}

	names, err := categoryNamesForUser(r.Context(), deps, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	renderNamed(w, r, deps, "import", "import_mapping", "", map[string]any{
		"Columns":           sheet.Columns,
		"Sample":            sheet.Sample,
		"Rows":              sheet.Rows,
		"Roles":             roles,
		"DateFormats":       csvimport.DateFormats,
		"DateLayout":        m.DateLayout,
		"NegativeIsExpense": m.NegativeIsExpense,
		"FallbackCategory":  m.FallbackCategory,
		"AmbiguousDate":     sheet.AmbiguousDate,
		"CategoryNames":     names,
		"Fingerprint":       sheet.Fingerprint,
		"Error":             errMsg,
	})
}

// categoryNamesForUser lists the account's categories the way they are
// displayed, which is what the fallback picker offers and what csvimport
// matches a name against.
func categoryNamesForUser(ctx context.Context, deps Deps, userID int64) ([]string, error) {
	rows, err := deps.Queries.ListCategoriesForUser(ctx, pgInt64(userID))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, c := range rows {
		names = append(names, i18n.CategoryName(c.Slug, c.Name))
	}
	return names, nil
}

// mappingFieldValues is the mapping as hidden form fields, so the preview
// and the confirm step can carry it back without the server keeping it.
func mappingFieldValues(m csvimport.Mapping) map[string]string {
	values := map[string]string{
		"mapped":            "1",
		"date_layout":       m.DateLayout,
		"fallback_category": m.FallbackCategory,
	}
	if m.NegativeIsExpense {
		values["negative_is_expense"] = "1"
	}
	for _, f := range mappingFields {
		copied := m
		values[f.Field] = strconv.Itoa(*f.get(&copied))
	}
	return values
}

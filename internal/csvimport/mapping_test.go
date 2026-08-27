package csvimport_test

import (
	"strings"
	"testing"

	"expensetracker/internal/csvimport"
)

// foreign is a mapping no export ever writes: the columns in another order,
// no type column at all, and a minus sign carrying the expense/income
// distinction on its own.
var foreign = csvimport.Mapping{
	Date: 1, Amount: 2, Category: 0, Note: 3, Type: csvimport.NoColumn,
	DateLayout: csvimport.DateDMYSlash, NegativeIsExpense: true,
}

func TestPlanFollowsTheMappingItIsGiven(t *testing.T) {
	const file = "Group,When,Value,Memo\n" +
		"Food & Drink,11/08/2026,-45000,Cà phê\n" +
		"Salary,01/08/2026,20000000,August\n"

	got, err := csvimport.Plan(strings.NewReader(file), foreign, catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("Plan() reported errors, want none: %+v", got.Errors)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("Plan() produced %d rows, want 2", len(got.Rows))
	}
	if first := got.Rows[0]; first.Type != "expense" || first.Amount != 45000 || first.CategoryID != 1 || first.Note != "Cà phê" {
		t.Errorf("Plan() first row = %+v, want a 45000 expense on category 1", first)
	}
	if second := got.Rows[1]; second.Type != "income" || second.Amount != 20000000 || second.CategoryID != 2 {
		t.Errorf("Plan() second row = %+v, want a 20000000 income on category 2", second)
	}
}

func TestPlanReadsAmountsWrittenTheWayExportersWriteThem(t *testing.T) {
	cases := []struct {
		in       string
		amount   int64
		wantType string
	}{
		{"45000", 45000, "income"},
		{"-45000", 45000, "expense"},
		{"45.000", 45000, "income"},
		{"45,000", 45000, "income"},
		{"45,000.00", 45000, "income"},
		{"45.000,00", 45000, "income"},
		{"45 000 ₫", 45000, "income"},
		{"(45000)", 45000, "expense"},
		{"45.50", 46, "income"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			// The amount is quoted because several of these cases contain a
			// comma, which is the field separator everywhere else.
			file := "Group,When,Value,Memo\nFood & Drink,11/08/2026,\"" + tc.in + "\",\n"
			got, err := csvimport.Plan(strings.NewReader(file), foreign, catalog, now)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if len(got.Rows) != 1 {
				t.Fatalf("Plan(%q) produced %d rows and errors %+v, want 1 row", tc.in, len(got.Rows), got.Errors)
			}
			if got.Rows[0].Amount != tc.amount {
				t.Errorf("Plan(%q) amount = %d, want %d", tc.in, got.Rows[0].Amount, tc.amount)
			}
			if got.Rows[0].Type != tc.wantType {
				t.Errorf("Plan(%q) type = %q, want %q", tc.in, got.Rows[0].Type, tc.wantType)
			}
		})
	}
}

func TestPlanReadsATypeColumnInEitherLanguage(t *testing.T) {
	mapping := csvimport.Mapping{
		Date: 0, Type: 1, Category: 2, Amount: 3, Note: csvimport.NoColumn,
		DateLayout: csvimport.DateISO,
	}
	const file = "Date,Kind,Group,Value\n" +
		"2026-08-11,chi,Food & Drink,45000\n" +
		"2026-08-01,Credit,Salary,20000000\n" +
		"2026-08-02,nonsense,Salary,10000\n"

	got, err := csvimport.Plan(strings.NewReader(file), mapping, catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("Plan() produced %d rows, want 2 (the third names no known type)", len(got.Rows))
	}
	if got.Rows[0].Type != "expense" || got.Rows[1].Type != "income" {
		t.Errorf("Plan() types = %q, %q, want expense, income", got.Rows[0].Type, got.Rows[1].Type)
	}
	if len(got.Errors) != 1 || got.Errors[0].Line != 4 {
		t.Errorf("Plan() errors = %+v, want one on line 4", got.Errors)
	}
}

func TestPlanFilesEverythingUnderTheFallbackWhenThereIsNoCategoryColumn(t *testing.T) {
	mapping := csvimport.Mapping{
		Date: 0, Amount: 1, Category: csvimport.NoColumn, Note: csvimport.NoColumn,
		Type: csvimport.NoColumn, DateLayout: csvimport.DateISO,
		NegativeIsExpense: true, FallbackCategory: "Food & Drink",
	}
	const file = "Date,Value\n2026-08-11,-45000\n2026-08-12,-50000\n"

	got, err := csvimport.Plan(strings.NewReader(file), mapping, catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(got.NewCategories) != 0 {
		t.Errorf("Plan() wants to create %+v, want it to reuse the fallback", got.NewCategories)
	}
	for _, row := range got.Rows {
		if row.CategoryID != 1 {
			t.Errorf("Plan() row on line %d = category %d, want the fallback's 1", row.Line, row.CategoryID)
		}
	}
}

package csvimport_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/csvimport"
)

// catalog is what the handler hands Plan: the categories the account can
// already spend against, defaults (which carry a slug) and its own (which
// do not).
var catalog = []csvimport.Category{
	{ID: 1, Name: "Food & Drink", Slug: "food_drink", Type: "expense"},
	{ID: 2, Name: "Salary", Slug: "salary", Type: "income"},
	{ID: 7, Name: "Du lịch", Type: "expense"},
}

var now = time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

func TestPlanReadsTheExportFormatItRoundTripsWith(t *testing.T) {
	// The BOM is what the export writes ahead of the header, so the header
	// match has to see through it or every round trip fails on line 1.
	const file = "\ufeffDate,Type,Category,Amount,Note\n" +
		"2026-08-11,expense,Food & Drink,45000,Cà phê\n" +
		"2026-08-01,income,Salary,20000000,\n"

	got, err := csvimport.Plan(strings.NewReader(file), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v, want nil", err)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("Plan() reported %d row errors, want none: %+v", len(got.Errors), got.Errors)
	}
	if len(got.NewCategories) != 0 {
		t.Errorf("Plan() wants to create %+v, want nothing new", got.NewCategories)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("Plan() produced %d rows, want 2", len(got.Rows))
	}

	first := got.Rows[0]
	want := csvimport.Row{
		Line: 2, Date: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
		Type: "expense", Amount: 45000, Note: "Cà phê", CategoryID: 1, CategoryName: "Food & Drink",
	}
	if first != want {
		t.Errorf("Plan() first row = %+v, want %+v", first, want)
	}
	if second := got.Rows[1]; second.CategoryID != 2 || second.Amount != 20000000 || second.Note != "" {
		t.Errorf("Plan() second row = %+v, want the salary row against category 2", second)
	}
}

func TestPlanRejectsBadLinesWithoutStoppingAtTheFirst(t *testing.T) {
	file := "Date,Type,Category,Amount,Note\n" +
		"11/08/2026,expense,Food & Drink,45000,\n" +
		"2026-08-11,transfer,Food & Drink,45000,\n" +
		"2026-08-11,expense,Food & Drink,\"45,000\",\n" +
		"2026-08-11,expense,Food & Drink,0,\n" +
		"2026-09-30,expense,Food & Drink,45000,\n" +
		"2026-08-11,expense,Food & Drink,45000," + strings.Repeat("x", 201) + "\n"

	got, err := csvimport.Plan(strings.NewReader(file), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v, want nil", err)
	}
	if len(got.Rows) != 0 {
		t.Errorf("Plan() accepted %d rows, want none: %+v", len(got.Rows), got.Rows)
	}
	wants := []struct {
		line  int
		about string
	}{
		{2, "YYYY-MM-DD"},
		{3, "expense or income"},
		{4, "separators"},
		{5, "above zero"},
		{6, "future"},
		{7, "200"},
	}
	if len(got.Errors) != len(wants) {
		t.Fatalf("Plan() reported %d errors, want %d: %+v", len(got.Errors), len(wants), got.Errors)
	}
	for i, want := range wants {
		if got.Errors[i].Line != want.line || !strings.Contains(got.Errors[i].Message, want.about) {
			t.Errorf("Plan() error %d = %+v, want line %d mentioning %q", i, got.Errors[i], want.line, want.about)
		}
	}
}

// TestPlanCreatesOneCategoryPerNameAndType mirrors the table's own
// (user_id, type, name) uniqueness: the same word on an expense line and an
// income line is two different categories, not one shared between them.
func TestPlanCreatesOneCategoryPerNameAndType(t *testing.T) {
	const file = "Date,Type,Category,Amount,Note\n" +
		"2026-08-11,expense,Cà phê,45000,\n" +
		"2026-08-12,expense,Cà phê,50000,\n" +
		"2026-08-13,income,Cà phê,900000,sold the beans\n" +
		"2026-08-14,expense,Du lịch,120000,\n"

	got, err := csvimport.Plan(strings.NewReader(file), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v, want nil", err)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("Plan() reported errors, want none: %+v", got.Errors)
	}
	want := []csvimport.NewCategory{
		{Name: "Cà phê", Type: "expense"},
		{Name: "Cà phê", Type: "income"},
	}
	if len(got.NewCategories) != len(want) {
		t.Fatalf("Plan() plans %+v, want %+v", got.NewCategories, want)
	}
	for i, w := range want {
		if got.NewCategories[i] != w {
			t.Errorf("Plan() new category %d = %+v, want %+v", i, got.NewCategories[i], w)
		}
	}
	// "Du lịch" is already the account's own expense category, so the row
	// resolves to it rather than planning a fourth.
	if last := got.Rows[3]; last.CategoryID != 7 {
		t.Errorf("Plan() row 4 CategoryID = %d, want 7", last.CategoryID)
	}
	for _, row := range got.Rows[:3] {
		if row.CategoryID != 0 {
			t.Errorf("Plan() row on line %d resolved to category %d, want 0 for a category not created yet", row.Line, row.CategoryID)
		}
	}
}

// TestPlanFingerprintsWhatItRead covers the check that keeps a preview
// honest: the confirm step re-uploads the file, and only an identical
// fingerprint proves it is the file the preview described.
func TestPlanFingerprintsWhatItRead(t *testing.T) {
	const file = "Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45000,\n"
	const edited = "Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45001,\n"

	first, err := csvimport.Plan(strings.NewReader(file), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	again, err := csvimport.Plan(strings.NewReader(file), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	other, err := csvimport.Plan(strings.NewReader(edited), catalog, now)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if first.Fingerprint == "" {
		t.Fatal("Plan() left the fingerprint empty")
	}
	if first.Fingerprint != again.Fingerprint {
		t.Errorf("Plan() fingerprinted the same bytes as %q then %q, want one value", first.Fingerprint, again.Fingerprint)
	}
	if first.Fingerprint == other.Fingerprint {
		t.Errorf("Plan() gave an edited file the same fingerprint %q", other.Fingerprint)
	}
}

func TestPlanRefusesMoreRowsThanItWillImportInOneGo(t *testing.T) {
	var b strings.Builder
	b.WriteString("Date,Type,Category,Amount,Note\n")
	for i := 0; i <= csvimport.MaxRows; i++ {
		b.WriteString("2026-08-11,expense,Food & Drink,45000,\n")
	}

	_, err := csvimport.Plan(strings.NewReader(b.String()), catalog, now)
	if !errors.Is(err, csvimport.ErrTooManyRows) {
		t.Errorf("Plan(%d rows) error = %v, want ErrTooManyRows", csvimport.MaxRows+1, err)
	}
}

func TestPlanRefusesAHeaderFromSomeOtherApp(t *testing.T) {
	const file = "Ngày,Danh mục,Số tiền\n2026-08-11,Ăn uống,45000\n"

	if _, err := csvimport.Plan(strings.NewReader(file), catalog, now); err == nil {
		t.Error("Plan(foreign header) error = nil, want a header error")
	}
}

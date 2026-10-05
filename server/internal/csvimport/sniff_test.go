package csvimport_test

import (
	"strings"
	"testing"

	"expensetracker/internal/csvimport"
)

func TestSniffReadsTheShapeOfAForeignFile(t *testing.T) {
	const file = "Transaction Date,Description,Amount,Category\n" +
		"25/12/2026,Coffee,-45000,Food\n" +
		"26/12/2026,Salary,20000000,Income\n"

	sheet, err := csvimport.Sniff(strings.NewReader(file))
	if err != nil {
		t.Fatalf("Sniff() error = %v, want nil", err)
	}
	if got, want := sheet.Columns, []string{"Transaction Date", "Description", "Amount", "Category"}; !equalStrings(got, want) {
		t.Errorf("Sniff() columns = %v, want %v", got, want)
	}
	if sheet.Rows != 2 {
		t.Errorf("Sniff() counted %d rows, want 2", sheet.Rows)
	}
	if len(sheet.Sample) != 2 {
		t.Errorf("Sniff() sampled %d rows, want 2", len(sheet.Sample))
	}
	if sheet.Fingerprint == "" {
		t.Error("Sniff() left the fingerprint empty")
	}
	if sheet.Exact {
		t.Error("Sniff() called a foreign header the export's own format")
	}

	guess := sheet.Guess
	if guess.Date != 0 || guess.Note != 1 || guess.Amount != 2 || guess.Category != 3 {
		t.Errorf("Sniff() guessed %+v, want date 0, note 1, amount 2, category 3", guess)
	}
	if guess.Type != -1 {
		t.Errorf("Sniff() guessed a type column at %d, want none (-1)", guess.Type)
	}
	// 25/12 can only be day-first, and the negative amount can only mean an
	// expense in a file with no type column.
	if guess.DateLayout != csvimport.DateDMYSlash {
		t.Errorf("Sniff() guessed date layout %q, want %q", guess.DateLayout, csvimport.DateDMYSlash)
	}
	if !guess.NegativeIsExpense {
		t.Error("Sniff() did not notice the negative amounts")
	}
}

func TestSniffReadsVietnameseHeadersWithAndWithoutTones(t *testing.T) {
	for _, header := range []string{
		"Ngày,Loại,Danh mục,Số tiền,Ghi chú",
		"Ngay,Loai,Danh muc,So tien,Ghi chu",
	} {
		sheet, err := csvimport.Sniff(strings.NewReader(header + "\n2026-08-11,chi,Ăn uống,45000,Cà phê\n"))
		if err != nil {
			t.Fatalf("Sniff(%q) error = %v", header, err)
		}
		guess := sheet.Guess
		if guess.Date != 0 || guess.Type != 1 || guess.Category != 2 || guess.Amount != 3 || guess.Note != 4 {
			t.Errorf("Sniff(%q) guessed %+v, want the columns in file order", header, guess)
		}
	}
}

// TestSniffRecognisesTheAppsOwnExport is what keeps the round trip two
// clicks: a file this app wrote needs no mapping screen at all.
func TestSniffRecognisesTheAppsOwnExport(t *testing.T) {
	const file = "\ufeffDate,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45000,Cà phê\n"

	sheet, err := csvimport.Sniff(strings.NewReader(file))
	if err != nil {
		t.Fatalf("Sniff() error = %v", err)
	}
	if !sheet.Exact {
		t.Fatal("Sniff() did not recognise the export's own header")
	}
	want := csvimport.ExportMapping()
	if sheet.Guess != want {
		t.Errorf("Sniff() guessed %+v for its own export, want %+v", sheet.Guess, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

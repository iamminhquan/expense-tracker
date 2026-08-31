package handlers

import (
	"testing"
	"time"

	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
)

// duplicateTestRow builds a txnRow with just the fields markDuplicates looks
// at (date, amount, type) plus an id so a test can tell rows apart in a
// failure message. showYear/duplicate are left at their zero values, which
// is what every row starts at before markDuplicates runs.
func duplicateTestRow(id int64, date string, amount int64, typ string) txnRow {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		panic(err) // test fixture only ever gets a literal in the caller
	}
	return txnRow{ListTransactionsForMonthRow: sqlcgen.ListTransactionsForMonthRow{
		ID: id, OccurredOn: pgval.Date(d), Amount: amount, Type: typ,
	}}
}

// TestMarkDuplicatesFlagsRowsSharingDateAmountAndType is the case the mark
// exists for: a transaction typed by hand and the same transaction arriving
// again via the bank email, both on the page at once.
func TestMarkDuplicatesFlagsRowsSharingDateAmountAndType(t *testing.T) {
	rows := []txnRow{
		duplicateTestRow(1, "2026-08-30", 20000, "expense"),
		duplicateTestRow(2, "2026-08-30", 20000, "expense"),
	}
	markDuplicates(rows)
	for _, r := range rows {
		if !r.IsDuplicate() {
			t.Errorf("markDuplicates(rows) row %d IsDuplicate() = false, want true", r.ID)
		}
	}
}

// TestMarkDuplicatesLeavesDifferentDatesUnmarked covers the plan's second
// required case directly: two rows that differ only by date must not be
// flagged, since they are not the same transaction seen twice.
func TestMarkDuplicatesLeavesDifferentDatesUnmarked(t *testing.T) {
	rows := []txnRow{
		duplicateTestRow(1, "2026-08-29", 20000, "expense"),
		duplicateTestRow(2, "2026-08-30", 20000, "expense"),
	}
	markDuplicates(rows)
	for _, r := range rows {
		if r.IsDuplicate() {
			t.Errorf("markDuplicates(rows) row %d IsDuplicate() = true, want false", r.ID)
		}
	}
}

// TestMarkDuplicatesComparesAmountAndTypeToo rounds out the three fields the
// mark is defined on: two rows on the same date are not "the same
// transaction twice" just because the date matches -- the amount and the
// type have to match as well, or an unrelated expense and income on the same
// day would get flagged against each other.
func TestMarkDuplicatesComparesAmountAndTypeToo(t *testing.T) {
	tests := []struct {
		name string
		rows []txnRow
	}{
		{"different amount", []txnRow{
			duplicateTestRow(1, "2026-08-30", 20000, "expense"),
			duplicateTestRow(2, "2026-08-30", 30000, "expense"),
		}},
		{"different type", []txnRow{
			duplicateTestRow(1, "2026-08-30", 20000, "expense"),
			duplicateTestRow(2, "2026-08-30", 20000, "income"),
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			markDuplicates(tc.rows)
			for _, r := range tc.rows {
				if r.IsDuplicate() {
					t.Errorf("markDuplicates(rows) row %d IsDuplicate() = true, want false (%s)", r.ID, tc.name)
				}
			}
		})
	}
}

// TestMarkDuplicatesFlagsAllOfAGroupLargerThanTwo checks the grouping is not
// hardcoded to pairs: three matching rows must all end up flagged, not just
// the first two found.
func TestMarkDuplicatesFlagsAllOfAGroupLargerThanTwo(t *testing.T) {
	rows := []txnRow{
		duplicateTestRow(1, "2026-08-30", 20000, "expense"),
		duplicateTestRow(2, "2026-08-30", 20000, "expense"),
		duplicateTestRow(3, "2026-08-30", 20000, "expense"),
	}
	markDuplicates(rows)
	for _, r := range rows {
		if !r.IsDuplicate() {
			t.Errorf("markDuplicates(rows) row %d IsDuplicate() = false, want true", r.ID)
		}
	}
}

// TestMarkDuplicatesLeavesAUniqueRowInAMixedListUnmarked checks that a lone
// row sitting alongside an unrelated duplicate pair is not swept up with
// them -- markDuplicates groups by key, not by "any duplicate exists on the
// page".
func TestMarkDuplicatesLeavesAUniqueRowInAMixedListUnmarked(t *testing.T) {
	rows := []txnRow{
		duplicateTestRow(1, "2026-08-30", 20000, "expense"),
		duplicateTestRow(2, "2026-08-30", 20000, "expense"),
		duplicateTestRow(3, "2026-08-30", 99000, "expense"),
	}
	markDuplicates(rows)
	if !rows[0].IsDuplicate() || !rows[1].IsDuplicate() {
		t.Fatal("markDuplicates(rows) the matching pair should both be IsDuplicate() = true")
	}
	if rows[2].IsDuplicate() {
		t.Errorf("markDuplicates(rows) row %d IsDuplicate() = true, want false: it matches no other row", rows[2].ID)
	}
}

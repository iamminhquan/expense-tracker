package handlers

import (
	"testing"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestComparisonText(t *testing.T) {
	if got := comparisonText(0, 0, false); got != "No data for last month" {
		t.Errorf("comparisonText(no prev data) = %q", got)
	}
	if got := comparisonText(8800000, 10000000, true); got != "Last month 10,000,000₫ · down 12%" {
		t.Errorf("comparisonText(decrease) = %q", got)
	}
	if got := comparisonText(11000000, 10000000, true); got != "Last month 10,000,000₫ · up 10%" {
		t.Errorf("comparisonText(increase) = %q", got)
	}
}

// The mobile Spent/Earned cards sit side by side under the balance card now
// rather than spanning the screen one per row, so the comparison line has
// roughly half the width it used to and drops everything but the direction
// and the number.
func TestComparisonTextMobile(t *testing.T) {
	if got := comparisonTextMobile(0, 0, false); got != "No data" {
		t.Errorf("comparisonTextMobile(no prev data) = %q", got)
	}
	if got := comparisonTextMobile(8800000, 10000000, true); got != "Down 12%" {
		t.Errorf("comparisonTextMobile(decrease) = %q", got)
	}
	if got := comparisonTextMobile(11000000, 10000000, true); got != "Up 10%" {
		t.Errorf("comparisonTextMobile(increase) = %q", got)
	}
	if got := comparisonTextMobile(10000000, 10000000, true); got != "Unchanged" {
		t.Errorf("comparisonTextMobile(no change) = %q", got)
	}
}

func TestBuildPieDataCapsAtSixPlusOther(t *testing.T) {
	var breakdown []sqlcgen.CategoryBreakdownRow
	for i := 0; i < 8; i++ {
		breakdown = append(breakdown, sqlcgen.CategoryBreakdownRow{
			CategoryName: "Cat", CategoryColor: "#D97757", Total: int64(100 - i),
		})
	}
	labels, values, colors, legend := buildPieData(breakdown, 700)
	if len(labels) != 7 {
		t.Fatalf("expected 7 pie slices (6 + Other), got %d", len(labels))
	}
	if labels[6] != "Other" || colors[6] != "#A1A1AA" {
		t.Fatalf("expected the 7th slice to be the reserved-gray Other aggregate, got %q/%q", labels[6], colors[6])
	}
	// The aggregate slice and the real "other" category must read the same;
	// two different words for the same idea in one chart is a bug.
	if legend[6].Name != i18n.NameForSlug("other") {
		t.Fatalf("expected the aggregate legend entry to reuse the other-category label, got %q", legend[6].Name)
	}
	if len(legend) != 7 || len(values) != 7 {
		t.Fatalf("expected 7 legend entries and values, got legend=%d values=%d", len(legend), len(values))
	}
}

func TestBuildBarSeriesZeroPadsMissingMonths(t *testing.T) {
	current := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	series := []sqlcgen.MonthlyTotalsSeriesRow{
		{Month: pgtype.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Valid: true}, TotalExpense: 5000, TotalIncome: 9000},
	}
	labels, expense, income := buildBarSeries(series, current, 4)
	if len(labels) != 4 || len(expense) != 4 || len(income) != 4 {
		t.Fatalf("expected 4 months of series data, got %d labels", len(labels))
	}
	if labels[3] != "Aug" {
		t.Fatalf("expected the last label to be the current month (Aug), got %q", labels[3])
	}
	if expense[3] != 5000 || income[3] != 9000 {
		t.Fatalf("expected the current month's totals to carry through, got expense=%d income=%d", expense[3], income[3])
	}
	if expense[0] != 0 || income[0] != 0 {
		t.Fatalf("expected a month with no data to zero-pad, got expense=%d income=%d", expense[0], income[0])
	}
}

// The reported prod bug: a month whose expenses span the real "Other"
// default plus enough categories to push one past the top-N cut drew two
// slices that were both labelled "Other" and both the reserved gray,
// because the row and the synthetic aggregate never met each other. The
// tail's money belongs in the same bucket the real category already is.
func TestBuildPieDataFoldsTheOtherCategoryIntoTheAggregate(t *testing.T) {
	otherSlug := pgtype.Text{String: "other", Valid: true}
	breakdown := []sqlcgen.CategoryBreakdownRow{
		{CategoryName: "Travel", CategoryColor: "#D97AA0", Total: 1008000},
		{CategorySlug: pgtype.Text{String: "food_drink", Valid: true}, CategoryName: "Food & Drink", CategoryColor: "#D97757", Total: 770000},
		{CategorySlug: pgtype.Text{String: "shopping", Valid: true}, CategoryName: "Shopping", CategoryColor: "#8B7BD8", Total: 490000},
		{CategorySlug: pgtype.Text{String: "entertainment", Valid: true}, CategoryName: "Entertainment", CategoryColor: "#E0A82E", Total: 407000},
		{CategorySlug: pgtype.Text{String: "transport", Valid: true}, CategoryName: "Transport", CategoryColor: "#5B8DEF", Total: 340000},
		{CategorySlug: otherSlug, CategoryName: "Other", CategoryColor: "#A1A1AA", Total: 275000},
		{CategorySlug: pgtype.Text{String: "bills", Valid: true}, CategoryName: "Bills", CategoryColor: "#6BA292", Total: 137200},
	}
	labels, values, _, legend := buildPieData(breakdown, 3427200)

	otherCount := 0
	for _, label := range labels {
		if label == i18n.NameForSlug("other") {
			otherCount++
		}
	}
	if otherCount != 1 {
		t.Fatalf("buildPieData(7 rows incl. the other category) drew %d %q slices, want 1: %v", otherCount, i18n.NameForSlug("other"), labels)
	}
	// Pulling the other-category row out of the ranking frees the slot the
	// tail was being hidden in, so every real category keeps its own slice.
	if len(labels) != 7 {
		t.Fatalf("buildPieData(7 rows) = %d slices, want 7: %v", len(labels), labels)
	}
	for _, want := range []string{"Travel", "Bills"} {
		found := false
		for _, label := range labels {
			if label == want {
				found = true
			}
		}
		if !found {
			t.Errorf("buildPieData(7 rows) lost the %q slice: %v", want, labels)
		}
	}
	if got, want := values[6], int64(275000); got != want {
		t.Errorf("buildPieData(7 rows) other slice = %d, want %d", got, want)
	}
	if got, want := legend[6].Amount, vnd(275000); got != want {
		t.Errorf("buildPieData(7 rows) other legend amount = %q, want %q", got, want)
	}
}

// With more categories than slices, the tail joins the real other-category
// row in the one bucket rather than doubling it.
func TestBuildPieDataSumsTailIntoTheOtherCategory(t *testing.T) {
	breakdown := []sqlcgen.CategoryBreakdownRow{
		{CategorySlug: pgtype.Text{String: "other", Valid: true}, CategoryName: "Other", CategoryColor: "#A1A1AA", Total: 500},
	}
	for i := 0; i < 8; i++ {
		breakdown = append(breakdown, sqlcgen.CategoryBreakdownRow{
			CategoryName: "Cat", CategoryColor: "#D97757", Total: int64(100 - i),
		})
	}
	labels, values, colors, _ := buildPieData(breakdown, 1272)

	if len(labels) != 7 {
		t.Fatalf("buildPieData(9 rows) = %d slices, want 7: %v", len(labels), labels)
	}
	if labels[6] != i18n.NameForSlug("other") || colors[6] != "#A1A1AA" {
		t.Fatalf("buildPieData(9 rows) last slice = %q/%q, want the reserved-gray other bucket", labels[6], colors[6])
	}
	// 500 from the real category + the two rows past the cut (94 + 93).
	if got, want := values[6], int64(687); got != want {
		t.Errorf("buildPieData(9 rows) other slice = %d, want %d", got, want)
	}
}

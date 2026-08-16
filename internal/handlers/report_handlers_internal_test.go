package handlers

import (
	"testing"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestComparisonText(t *testing.T) {
	if got := comparisonText(0, 0, false); got != "Chưa có dữ liệu tháng trước" {
		t.Errorf("comparisonText(no prev data) = %q", got)
	}
	if got := comparisonText(8800000, 10000000, true); got != "Tháng trước 10.000.000₫ · giảm 12%" {
		t.Errorf("comparisonText(decrease) = %q", got)
	}
	if got := comparisonText(11000000, 10000000, true); got != "Tháng trước 10.000.000₫ · tăng 10%" {
		t.Errorf("comparisonText(increase) = %q", got)
	}
}

func TestComparisonTextMobile(t *testing.T) {
	if got := comparisonTextMobile(0, 0, false); got != "Chưa có dữ liệu tháng trước" {
		t.Errorf("comparisonTextMobile(no prev data) = %q", got)
	}
	if got := comparisonTextMobile(8800000, 10000000, true); got != "Giảm 12% so với tháng trước" {
		t.Errorf("comparisonTextMobile(decrease) = %q", got)
	}
	if got := comparisonTextMobile(11000000, 10000000, true); got != "Tăng 10% so với tháng trước" {
		t.Errorf("comparisonTextMobile(increase) = %q", got)
	}
	if got := comparisonTextMobile(10000000, 10000000, true); got != "Không đổi so với tháng trước" {
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
	labels, chi, thu := buildBarSeries(series, current, 4)
	if len(labels) != 4 || len(chi) != 4 || len(thu) != 4 {
		t.Fatalf("expected 4 months of series data, got %d labels", len(labels))
	}
	if labels[3] != "Th 8" {
		t.Fatalf("expected the last label to be the current month (Th 8), got %q", labels[3])
	}
	if chi[3] != 5000 || thu[3] != 9000 {
		t.Fatalf("expected the current month's totals to carry through, got chi=%d thu=%d", chi[3], thu[3])
	}
	if chi[0] != 0 || thu[0] != 0 {
		t.Fatalf("expected a month with no data to zero-pad, got chi=%d thu=%d", chi[0], thu[0])
	}
}

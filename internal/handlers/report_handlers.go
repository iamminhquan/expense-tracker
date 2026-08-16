package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/i18n"
	"expensetracker/internal/sqlcgen"
)

const pieTopN = 6
const barMonths = 4

type pieLegendEntry struct {
	Name    string
	Color   string
	Percent string
	Amount  string
}

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		data, err := buildDashboardData(r, deps, userID, r.URL.Query().Get("thang"))
		if err != nil {
			http.Error(w, "could not load dashboard", http.StatusInternalServerError)
			return
		}

		if isFragmentRequest(r) {
			renderNamed(w, r, deps, "dashboard", "dashboard_month_section", "", data)
			return
		}
		render(w, r, deps, "dashboard", "dashboard", data)
	}
}

func buildDashboardData(r *http.Request, deps Deps, userID int64, monthParam string) (map[string]any, error) {
	from, to := monthRangeFor(monthParam)

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}

	prevFrom := pgDate(from.Time.AddDate(0, -1, 0))
	prevTotals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: prevFrom, OccurredOn_2: from})
	if err != nil {
		return nil, err
	}
	hasPrevData := prevTotals.TotalExpense > 0 || prevTotals.TotalIncome > 0

	breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	pieLabels, pieValues, pieColors, legend := buildPieData(breakdown, totals.TotalExpense)

	seriesFrom := pgDate(from.Time.AddDate(0, -(barMonths - 1), 0))
	series, err := deps.Queries.MonthlyTotalsSeries(r.Context(), sqlcgen.MonthlyTotalsSeriesParams{UserID: userID, OccurredOn: seriesFrom, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	barLabels, barChi, barThu := buildBarSeries(series, from.Time, barMonths)
	hasAnyMonthData := false
	for i := range barChi {
		if barChi[i] > 0 || barThu[i] > 0 {
			hasAnyMonthData = true
		}
	}

	months, err := deps.Queries.ListDistinctTransactionMonths(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	currentFrom, _ := currentMonthRange()
	var available []map[string]any
	for _, m := range months {
		if m.Time.Year() == currentFrom.Time.Year() && m.Time.Month() == currentFrom.Time.Month() {
			continue
		}
		available = append(available, map[string]any{"Value": m.Time.Format("2006-01"), "Label": monthLabel(m.Time)})
	}

	pieLabelsJSON, _ := json.Marshal(pieLabels)
	pieValuesJSON, _ := json.Marshal(pieValues)
	pieColorsJSON, _ := json.Marshal(pieColors)
	barLabelsJSON, _ := json.Marshal(barLabels)
	barChiJSON, _ := json.Marshal(barChi)
	barThuJSON, _ := json.Marshal(barThu)

	return map[string]any{
		"MonthLabel":              monthLabel(from.Time),
		"CurrentMonthValue":       currentFrom.Time.Format("2006-01"),
		"AvailableMonths":         available,
		"TotalExpense":            totals.TotalExpense,
		"TotalIncome":             totals.TotalIncome,
		"ExpenseComparison":       comparisonText(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
		"IncomeComparison":        comparisonText(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
		"ExpenseComparisonMobile": comparisonTextMobile(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
		"IncomeComparisonMobile":  comparisonTextMobile(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
		"CurrentMonthEmpty":       totals.TotalExpense == 0 && totals.TotalIncome == 0,
		"HasAnyMonthData":         hasAnyMonthData,
		"PieLegend":               legend,
		"PieLabelsJSON":           template.JS(pieLabelsJSON),
		"PieValuesJSON":           template.JS(pieValuesJSON),
		"PieColorsJSON":           template.JS(pieColorsJSON),
		"BarLabelsJSON":           template.JS(barLabelsJSON),
		"BarChiJSON":              template.JS(barChiJSON),
		"BarThuJSON":              template.JS(barThuJSON),
	}, nil
}

// buildPieData turns CategoryBreakdown's already-total-desc-ordered rows
// into the top pieTopN slices plus a synthetic "Other" aggregate for
// everything past that, per SPEC.md section 5.
func buildPieData(breakdown []sqlcgen.CategoryBreakdownRow, totalExpense int64) (labels []string, values []int64, colors []string, legend []pieLegendEntry) {
	shown := breakdown
	var otherSum int64
	if len(breakdown) > pieTopN {
		shown = breakdown[:pieTopN]
		for _, row := range breakdown[pieTopN:] {
			otherSum += row.Total
		}
	}
	for _, row := range shown {
		// Resolved here rather than in the template: the same name goes
		// into the legend (HTML) and into labels, which is serialised to
		// JSON for Chart.js, where a template func cannot reach it.
		name := i18n.CategoryName(row.CategorySlug, row.CategoryName)
		labels = append(labels, name)
		values = append(values, row.Total)
		colors = append(colors, row.CategoryColor)
		legend = append(legend, pieLegendEntry{
			Name: name, Color: row.CategoryColor,
			Percent: percentOf(row.Total, totalExpense), Amount: vnd(row.Total),
		})
	}
	if otherSum > 0 {
		// The same label the real "other" category uses, so the aggregate
		// slice and that category never read as two different things.
		otherName := i18n.NameForSlug("other")
		labels = append(labels, otherName)
		values = append(values, otherSum)
		colors = append(colors, "#A1A1AA")
		legend = append(legend, pieLegendEntry{
			Name: otherName, Color: "#A1A1AA",
			Percent: percentOf(otherSum, totalExpense), Amount: vnd(otherSum),
		})
	}
	return
}

func percentOf(part, total int64) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", int(float64(part)/float64(total)*100+0.5))
}

// buildBarSeries returns exactly `months` consecutive [oldest..newest]
// points ending at currentMonthStart, zero-padding any month
// MonthlyTotalsSeries didn't return a row for.
func buildBarSeries(series []sqlcgen.MonthlyTotalsSeriesRow, currentMonthStart time.Time, months int) (labels []string, chi []int64, thu []int64) {
	byMonth := make(map[string]sqlcgen.MonthlyTotalsSeriesRow, len(series))
	for _, row := range series {
		byMonth[row.Month.Time.Format("2006-01")] = row
	}
	for i := months - 1; i >= 0; i-- {
		m := currentMonthStart.AddDate(0, -i, 0)
		key := m.Format("2006-01")
		labels = append(labels, shortMonthLabel(m))
		if row, ok := byMonth[key]; ok {
			chi = append(chi, row.TotalExpense)
			thu = append(thu, row.TotalIncome)
		} else {
			chi = append(chi, 0)
			thu = append(thu, 0)
		}
	}
	return
}

func shortMonthLabel(t time.Time) string {
	return fmt.Sprintf("Th %d", int(t.Month()))
}

// comparisonText builds SPEC.md section 5's "Tháng trước X · tăng/giảm Y%"
// line, or its "no data" fallback when the previous month had zero
// transactions of any kind.
func comparisonText(current, previous int64, hasPrevData bool) string {
	if !hasPrevData {
		return "Chưa có dữ liệu tháng trước"
	}
	if previous == 0 {
		return fmt.Sprintf("Tháng trước %s", vnd(previous))
	}
	diff := current - previous
	// Rounds to the nearest percent (matching percentOf's rounding above)
	// rather than truncating, so e.g. a 12.6% change reads as "13%" here
	// too, not "12%".
	pct := int(math.Abs(float64(diff))/float64(previous)*100 + 0.5)
	if diff == 0 {
		return fmt.Sprintf("Tháng trước %s · không đổi", vnd(previous))
	}
	direction := "tăng"
	if diff < 0 {
		direction = "giảm"
	}
	return fmt.Sprintf("Tháng trước %s · %s %d%%", vnd(previous), direction, pct)
}

// comparisonTextMobile builds SPEC.md section 5's mobile-only shortened
// comparison line (e.g. "Giảm 11% so với tháng trước"), shown below 768px
// in place of comparisonText's fuller "Tháng trước X · giảm Y%" -- the
// amount is dropped to fit the narrower stat card.
func comparisonTextMobile(current, previous int64, hasPrevData bool) string {
	if !hasPrevData || previous == 0 {
		return "Chưa có dữ liệu tháng trước"
	}
	diff := current - previous
	if diff == 0 {
		return "Không đổi so với tháng trước"
	}
	pct := int(math.Abs(float64(diff))/float64(previous)*100 + 0.5)
	direction := "Tăng"
	if diff < 0 {
		direction = "Giảm"
	}
	return fmt.Sprintf("%s %d%% so với tháng trước", direction, pct)
}

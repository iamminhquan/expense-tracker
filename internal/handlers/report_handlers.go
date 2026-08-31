package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
)

const pieTopN = 6
const barMonths = 4

// otherSlug identifies the catch-all default category. Matched on the slug
// rather than the displayed name, which a translation would move.
const otherSlug = "other"

type pieLegendEntry struct {
	Name    string
	Color   string
	Percent string
	Amount  string
}

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		data, err := buildDashboardData(r, deps, userID, r.URL.Query().Get("month"))
		if err != nil {
			http.Error(w, "could not load dashboard", http.StatusInternalServerError)
			return
		}

		if isFragmentRequest(r) {
			renderViewNamed(w, r, deps, "dashboard", "dashboard_month_section", "dashboard", data)
			return
		}
		renderView(w, r, deps, "dashboard", "dashboard", data)
	}
}

// dashboardView is the whole dashboard: the month it is showing, the two
// totals with their comparison lines, and the two charts' data already
// serialised, since Chart.js reads them out of the page as JSON rather than
// from anything the template could build.
//
// The comparison lines come in two spellings because the mobile cards share
// a row and have half the width to say the same thing in.
type dashboardView struct {
	viewData

	MonthLabel        string
	CurrentMonthValue string
	AvailableMonths   []monthOption

	TotalExpense            int64
	TotalIncome             int64
	ExpenseComparison       string
	IncomeComparison        string
	ExpenseComparisonMobile string
	IncomeComparisonMobile  string
	CurrentMonthEmpty       bool
	HasAnyMonthData         bool

	PieLegend      []pieLegendEntry
	PieLabelsJSON  template.JS
	PieValuesJSON  template.JS
	PieColorsJSON  template.JS
	BarLabelsJSON  template.JS
	BarExpenseJSON template.JS
	BarIncomeJSON  template.JS
}

func buildDashboardData(r *http.Request, deps Deps, userID int64, monthParam string) (*dashboardView, error) {
	from, to := monthRangeFor(monthParam)

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}

	prevFrom := pgval.Date(from.Time.AddDate(0, -1, 0))
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

	seriesFrom := pgval.Date(from.Time.AddDate(0, -(barMonths - 1), 0))
	series, err := deps.Queries.MonthlyTotalsSeries(r.Context(), sqlcgen.MonthlyTotalsSeriesParams{UserID: userID, OccurredOn: seriesFrom, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	barLabels, barExpense, barIncome := buildBarSeries(series, from.Time, barMonths)
	hasAnyMonthData := false
	for i := range barExpense {
		if barExpense[i] > 0 || barIncome[i] > 0 {
			hasAnyMonthData = true
		}
	}

	months, err := deps.Queries.ListDistinctTransactionMonths(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	currentFrom, _ := currentMonthRange()

	pieLabelsJSON, _ := json.Marshal(pieLabels)
	pieValuesJSON, _ := json.Marshal(pieValues)
	pieColorsJSON, _ := json.Marshal(pieColors)
	barLabelsJSON, _ := json.Marshal(barLabels)
	barExpenseJSON, _ := json.Marshal(barExpense)
	barIncomeJSON, _ := json.Marshal(barIncome)

	return &dashboardView{
		MonthLabel:              monthLabel(from.Time),
		CurrentMonthValue:       currentFrom.Time.Format("2006-01"),
		AvailableMonths:         monthOptions(months, currentFrom),
		TotalExpense:            totals.TotalExpense,
		TotalIncome:             totals.TotalIncome,
		ExpenseComparison:       comparisonText(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
		IncomeComparison:        comparisonText(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
		ExpenseComparisonMobile: comparisonTextMobile(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
		IncomeComparisonMobile:  comparisonTextMobile(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
		CurrentMonthEmpty:       totals.TotalExpense == 0 && totals.TotalIncome == 0,
		HasAnyMonthData:         hasAnyMonthData,
		PieLegend:               legend,
		PieLabelsJSON:           template.JS(pieLabelsJSON),
		PieValuesJSON:           template.JS(pieValuesJSON),
		PieColorsJSON:           template.JS(pieColorsJSON),
		BarLabelsJSON:           template.JS(barLabelsJSON),
		BarExpenseJSON:          template.JS(barExpenseJSON),
		BarIncomeJSON:           template.JS(barIncomeJSON),
	}, nil
}

// buildPieData turns CategoryBreakdown's already-total-desc-ordered rows
// into the top pieTopN slices plus an "Other" aggregate for everything past
// that, so the doughnut never grows an unreadable tail of one-percent
// slivers.
//
// The real "other" default category never competes for one of those slices:
// it is the same bucket the aggregate describes, so it is lifted out of the
// ranking and the two are summed. Left in, a month holding both drew two
// slices that were identically named and identically gray -- the reserved
// #A1A1AA is that category's own color -- and, because the row occupied a
// slot, hid a real category behind the aggregate's label as well.
func buildPieData(breakdown []sqlcgen.CategoryBreakdownRow, totalExpense int64) (labels []string, values []int64, colors []string, legend []pieLegendEntry) {
	var ranked []sqlcgen.CategoryBreakdownRow
	var otherSum int64
	for _, row := range breakdown {
		if row.CategorySlug.Valid && row.CategorySlug.String == otherSlug {
			otherSum += row.Total
			continue
		}
		ranked = append(ranked, row)
	}
	shown := ranked
	if len(ranked) > pieTopN {
		shown = ranked[:pieTopN]
		for _, row := range ranked[pieTopN:] {
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
			Percent: percentOf(row.Total, totalExpense), Amount: format.VND(row.Total),
		})
	}
	if otherSum > 0 {
		// The same label the real "other" category uses, so the aggregate
		// slice and that category never read as two different things.
		otherName := i18n.NameForSlug(otherSlug)
		labels = append(labels, otherName)
		values = append(values, otherSum)
		colors = append(colors, "#A1A1AA")
		legend = append(legend, pieLegendEntry{
			Name: otherName, Color: "#A1A1AA",
			Percent: percentOf(otherSum, totalExpense), Amount: format.VND(otherSum),
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
func buildBarSeries(series []sqlcgen.MonthlyTotalsSeriesRow, currentMonthStart time.Time, months int) (labels []string, expense []int64, income []int64) {
	byMonth := make(map[string]sqlcgen.MonthlyTotalsSeriesRow, len(series))
	for _, row := range series {
		byMonth[row.Month.Time.Format("2006-01")] = row
	}
	for i := months - 1; i >= 0; i-- {
		m := currentMonthStart.AddDate(0, -i, 0)
		key := m.Format("2006-01")
		labels = append(labels, shortMonthLabel(m))
		if row, ok := byMonth[key]; ok {
			expense = append(expense, row.TotalExpense)
			income = append(income, row.TotalIncome)
		} else {
			expense = append(expense, 0)
			income = append(income, 0)
		}
	}
	return
}

func shortMonthLabel(t time.Time) string {
	return t.Format("Jan")
}

// comparisonText builds the "Last month X · up/down Y%" line under each
// dashboard total, or its "no data" fallback when the previous month had
// zero transactions of any kind.
func comparisonText(current, previous int64, hasPrevData bool) string {
	if !hasPrevData {
		return "No data for last month"
	}
	if previous == 0 {
		return fmt.Sprintf("Last month %s", format.VND(previous))
	}
	diff := current - previous
	// Rounds to the nearest percent (matching percentOf's rounding above)
	// rather than truncating, so e.g. a 12.6% change reads as "13%" here
	// too, not "12%".
	pct := int(math.Abs(float64(diff))/float64(previous)*100 + 0.5)
	if diff == 0 {
		return fmt.Sprintf("Last month %s · unchanged", format.VND(previous))
	}
	direction := "up"
	if diff < 0 {
		direction = "down"
	}
	return fmt.Sprintf("Last month %s · %s %d%%", format.VND(previous), direction, pct)
}

// comparisonTextMobile builds the shortened comparison line (e.g. "Down
// 11%") shown below 768px in place of comparisonText's fuller "Last month
// X · down Y%".
//
// It keeps only the direction and the percentage: since the balance card
// took the full width at the top of the dashboard, the Spent and Earned
// cards share a row beneath it and have roughly half the width they had
// when this line still ended in "vs last month".
func comparisonTextMobile(current, previous int64, hasPrevData bool) string {
	if !hasPrevData || previous == 0 {
		return "No data"
	}
	diff := current - previous
	if diff == 0 {
		return "Unchanged"
	}
	pct := int(math.Abs(float64(diff))/float64(previous)*100 + 0.5)
	direction := "Up"
	if diff < 0 {
		direction = "Down"
	}
	return fmt.Sprintf("%s %d%%", direction, pct)
}

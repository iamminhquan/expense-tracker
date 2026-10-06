package api

import (
	"net/http"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/gin-gonic/gin"
)

const pieTopN = 6
const barMonths = 4

// otherSlug mirrors handlers.otherSlug -- the catch-all default category,
// matched on slug rather than displayed name, which a translation would
// move.
const otherSlug = "other"

// balanceDTO mirrors handlers.balanceSummary, dropping RatioLabel: that was
// a pre-formatted sentence ("Spent 42% of this month's income") for a
// template that cannot itself compute one. A JSON client has SpentPct and
// HasIncome to build whatever sentence it wants, in whatever language it
// wants, without this API baking English prose into the response -- the
// same reasoning that kept pie/bar data as plain numbers below instead of
// handlers.go's template.JS-wrapped JSON strings.
//
// HasIncome exists because SpentPct alone is ambiguous at zero: "no income
// this month" and "income, but nothing spent of it yet" both leave
// SpentPct at its zero value, and a client showing the header widget (every
// page, not just the dashboard, which is the only place TotalIncome is
// otherwise visible) has no other way to tell the two apart.
type balanceDTO struct {
	Remaining int64 `json:"remaining"`
	SpentPct  int   `json:"spentPct"`
	HasIncome bool  `json:"hasIncome"`
	Empty     bool  `json:"empty"`
}

func newBalanceDTO(expense, income, carriedOver int64) balanceDTO {
	d := balanceDTO{
		Remaining: carriedOver + income - expense,
		HasIncome: income > 0,
		Empty:     carriedOver == 0 && expense == 0 && income == 0,
	}
	if income <= 0 {
		return d
	}
	if expense > income {
		d.SpentPct = 100
		return d
	}
	d.SpentPct = int(expense * 100 / income)
	return d
}

// pieLegendEntryDTO mirrors handlers.pieLegendEntry, but Percent is a plain
// int (not a pre-formatted "42%" string) and Amount is the raw int64 đồng
// figure (not format.VND's formatted string) -- see balanceDTO's comment;
// the client formats both.
type pieLegendEntryDTO struct {
	Name    string `json:"name"`
	Color   string `json:"color"`
	Percent int    `json:"percent"`
	Amount  int64  `json:"amount"`
}

type pieDataDTO struct {
	Labels []string            `json:"labels"`
	Values []int64             `json:"values"`
	Colors []string            `json:"colors"`
	Legend []pieLegendEntryDTO `json:"legend"`
}

type barDataDTO struct {
	Labels  []string `json:"labels"`
	Expense []int64  `json:"expense"`
	Income  []int64  `json:"income"`
}

type dashboardResponse struct {
	MonthValue        string           `json:"monthValue"`
	MonthLabel        string           `json:"monthLabel"`
	CurrentMonthValue string           `json:"currentMonthValue"`
	AvailableMonths   []monthOptionDTO `json:"availableMonths"`

	TotalExpense         int64 `json:"totalExpense"`
	TotalIncome          int64 `json:"totalIncome"`
	PreviousTotalExpense int64 `json:"previousTotalExpense"`
	PreviousTotalIncome  int64 `json:"previousTotalIncome"`
	HasPreviousMonthData bool  `json:"hasPreviousMonthData"`
	CurrentMonthEmpty    bool  `json:"currentMonthEmpty"`

	// HeaderBalance is always the real current month, regardless of
	// MonthValue -- the client-side equivalent of a persistent nav widget
	// that sits above any month picker. See
	// handlers.currentHeaderBalance's identical comment.
	HeaderBalance balanceDTO `json:"headerBalance"`

	Pie pieDataDTO `json:"pie"`
	Bar barDataDTO `json:"bar"`
}

func dashboardHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		ctx := c.Request.Context()
		from, to := monthRangeFor(c.Query("month"))

		totals, err := deps.Queries.MonthlyTotals(ctx, sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load dashboard")
			return
		}

		currentFrom, currentTo := currentMonthRange()
		var headerTotals sqlcgen.MonthlyTotalsRow
		if from.Time.Equal(currentFrom.Time) {
			headerTotals = totals
		} else {
			headerTotals, err = deps.Queries.MonthlyTotals(ctx, sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: currentFrom, OccurredOn_2: currentTo})
			if err != nil {
				respondError(c, http.StatusInternalServerError, "could not load dashboard")
				return
			}
		}

		prevFrom := pgval.Date(from.Time.AddDate(0, -1, 0))
		prevTotals, err := deps.Queries.MonthlyTotals(ctx, sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: prevFrom, OccurredOn_2: from})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load dashboard")
			return
		}
		hasPrevData := prevTotals.TotalExpense > 0 || prevTotals.TotalIncome > 0

		breakdown, err := deps.Queries.CategoryBreakdown(ctx, sqlcgen.CategoryBreakdownParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load dashboard")
			return
		}
		pie := buildPieData(breakdown, totals.TotalExpense)

		seriesFrom := pgval.Date(from.Time.AddDate(0, -(barMonths - 1), 0))
		series, err := deps.Queries.MonthlyTotalsSeries(ctx, sqlcgen.MonthlyTotalsSeriesParams{UserID: userID, OccurredOn: seriesFrom, OccurredOn_2: to})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load dashboard")
			return
		}
		bar := buildBarSeries(series, from.Time, barMonths)

		months, err := deps.Queries.ListDistinctTransactionMonths(ctx, userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load dashboard")
			return
		}

		respondOK(c, http.StatusOK, "dashboard retrieved", dashboardResponse{
			MonthValue:           from.Time.Format("2006-01"),
			MonthLabel:           monthLabel(from.Time),
			CurrentMonthValue:    currentFrom.Time.Format("2006-01"),
			AvailableMonths:      monthOptions(months, currentFrom),
			TotalExpense:         totals.TotalExpense,
			TotalIncome:          totals.TotalIncome,
			PreviousTotalExpense: prevTotals.TotalExpense,
			PreviousTotalIncome:  prevTotals.TotalIncome,
			HasPreviousMonthData: hasPrevData,
			CurrentMonthEmpty:    totals.TotalExpense == 0 && totals.TotalIncome == 0,
			HeaderBalance:        newBalanceDTO(headerTotals.TotalExpense, headerTotals.TotalIncome, headerTotals.CarriedOver),
			Pie:                  pie,
			Bar:                  bar,
		})
	}
}

// buildPieData mirrors handlers.buildPieData exactly -- see its comment
// for why the real "other" default category is lifted out of the ranking
// and summed into the synthetic aggregate slice rather than left to
// compete for one of the top pieTopN spots.
func buildPieData(breakdown []sqlcgen.CategoryBreakdownRow, totalExpense int64) pieDataDTO {
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

	// Every field starts as an empty (never nil) slice: encoding/json
	// renders a nil slice as JSON null rather than [], and a month with no
	// expenses at all -- the dashboard every brand-new account lands on --
	// hits exactly that (the loop below never runs, otherSum stays 0, so a
	// `var d pieDataDTO` zero value would ship every field as null). The
	// client indexes straight into these arrays (.length, .map) without a
	// null check, same as monthOptions and every other array field this
	// package returns -- that's the actual reason to follow this
	// convention everywhere, not just where it happens to matter today.
	d := pieDataDTO{Labels: []string{}, Values: []int64{}, Colors: []string{}, Legend: []pieLegendEntryDTO{}}
	for _, row := range shown {
		name := i18n.CategoryName(row.CategorySlug, row.CategoryName)
		d.Labels = append(d.Labels, name)
		d.Values = append(d.Values, row.Total)
		d.Colors = append(d.Colors, row.CategoryColor)
		d.Legend = append(d.Legend, pieLegendEntryDTO{
			Name: name, Color: row.CategoryColor,
			Percent: percentOf(row.Total, totalExpense), Amount: row.Total,
		})
	}
	if otherSum > 0 {
		otherName := i18n.NameForSlug(otherSlug)
		d.Labels = append(d.Labels, otherName)
		d.Values = append(d.Values, otherSum)
		d.Colors = append(d.Colors, "#A1A1AA")
		d.Legend = append(d.Legend, pieLegendEntryDTO{
			Name: otherName, Color: "#A1A1AA",
			Percent: percentOf(otherSum, totalExpense), Amount: otherSum,
		})
	}
	return d
}

func percentOf(part, total int64) int {
	if total == 0 {
		return 0
	}
	return int(float64(part)/float64(total)*100 + 0.5)
}

// buildBarSeries mirrors handlers.buildBarSeries exactly: exactly `months`
// consecutive [oldest..newest] points ending at currentMonthStart,
// zero-padding any month the series query didn't return a row for.
func buildBarSeries(series []sqlcgen.MonthlyTotalsSeriesRow, currentMonthStart time.Time, months int) barDataDTO {
	byMonth := make(map[string]sqlcgen.MonthlyTotalsSeriesRow, len(series))
	for _, row := range series {
		byMonth[row.Month.Time.Format("2006-01")] = row
	}
	// Always ends up with exactly `months` entries via the loop below, so
	// this can't actually go out nil -- initialized as empty slices anyway
	// to keep the convention uniform across every array field in this
	// package rather than leaving a reader to wonder why this one DTO is
	// the exception.
	d := barDataDTO{Labels: []string{}, Expense: []int64{}, Income: []int64{}}
	for i := months - 1; i >= 0; i-- {
		m := currentMonthStart.AddDate(0, -i, 0)
		d.Labels = append(d.Labels, m.Format("Jan"))
		if row, ok := byMonth[m.Format("2006-01")]; ok {
			d.Expense = append(d.Expense, row.TotalExpense)
			d.Income = append(d.Income, row.TotalIncome)
		} else {
			d.Expense = append(d.Expense, 0)
			d.Income = append(d.Income, 0)
		}
	}
	return d
}

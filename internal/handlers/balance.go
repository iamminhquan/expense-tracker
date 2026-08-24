package handlers

import "fmt"

// balanceSummary is the figure the nav header widget renders: the running
// balance, plus how the displayed month's spending measured against that
// month's own income.
//
// The two scales sit side by side on purpose. Remaining runs forward across
// months, while SpentPct and RatioLabel describe one month alone -- "spent
// 42% of this month's income" is the useful reading, and measuring a month's
// spending against a balance built up over years would report a steadily
// shrinking percentage that says nothing about how the month went.
//
// The percentages are resolved here rather than in the template because
// html/template cannot divide, and because every interesting case is a
// division that has to be guarded: a month with no income at all, and a
// month whose spending runs past the end of the bar.
type balanceSummary struct {
	// Remaining is the balance at the end of the displayed month: what
	// earlier months carried in, plus this month's income less its expenses.
	Remaining int64

	// SpentPct is the bar fill's width in percent, clamped to 100 so
	// overspending fills the bar rather than overflowing it.
	SpentPct   int
	RatioLabel string

	// Empty marks a widget with nothing at all to show -- no transactions
	// this month and no balance carried into it -- where the 0₫ is greyed to
	// ink-zero. A carried-in balance is something to show, so an untouched
	// month that follows an active one is not empty.
	Empty bool
}

// newBalanceSummary computes the running balance and the month's spending
// ratio from the three totals MonthlyTotals returns.
//
// The balance runs forward rather than resetting on the 1st: a month starts
// from whatever the ones before it left behind, so what a month closes at is
// exactly what the next one opens with.
//
// The arithmetic stays in int64 throughout: amounts are whole đồng, and
// going through float64 to compute a percentage would introduce rounding
// that the displayed number would eventually disagree with.
func newBalanceSummary(expense, income, carriedOver int64) balanceSummary {
	summary := balanceSummary{
		Remaining:  carriedOver + income - expense,
		Empty:      carriedOver == 0 && expense == 0 && income == 0,
		RatioLabel: "No income this month",
	}

	if income <= 0 {
		return summary
	}

	pct := int(expense * 100 / income)
	switch {
	case expense > income:
		summary.SpentPct = 100
		summary.RatioLabel = "Over this month's income"
	default:
		summary.SpentPct = pct
		summary.RatioLabel = fmt.Sprintf("Spent %d%% of this month's income", pct)
	}
	return summary
}

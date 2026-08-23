package handlers

import (
	"fmt"
	"time"
)

// balanceCard is everything the shared "balance_card" partial renders. The
// two pages that show it differ only in the Variant string, which the
// template branches on for sizing and for whether it draws its own
// Spent/Earned detail row (the dashboard has sibling cards for that).
//
// The ratio bar's percentages are resolved here rather than in the template
// because html/template cannot divide, and because every interesting case is
// a division that has to be guarded: a month with no income at all, and a
// month whose spending runs past the end of the bar.
type balanceCard struct {
	// Expense and Income are the displayed month's own two totals; Remaining
	// is the running balance at the end of it, which is CarriedOver plus the
	// month's net rather than the month's net alone.
	Expense, Income, Remaining int64

	// CarriedOver is everything the user banked before this month began:
	// income minus expense over their whole history up to the 1st. The card
	// names it under the headline so the balance can't be misread as this
	// month's earnings, and ShowCarriedOver hides that note when there is no
	// history behind the month to describe.
	CarriedOver     int64
	ShowCarriedOver bool

	Month   time.Time
	Variant string

	// SpentPct is the bar fill's width in percent, clamped to 100 so
	// overspending fills the bar rather than overflowing it. LeftPct is the
	// remainder, shown only when ShowLeftPct -- "58% left" is meaningless
	// once spending has passed income, or when there was no income to
	// measure against.
	SpentPct    int
	LeftPct     int
	ShowLeftPct bool

	Label      string
	RatioLabel string

	// Empty marks a card with nothing at all to show -- no transactions this
	// month and no balance carried into it -- where the 0₫ is greyed to
	// ink-zero. A carried-in balance is something to show, so an untouched
	// month that follows an active one is not empty.
	Empty bool

	// OOB makes the partial tag itself hx-swap-oob. Only mutation responses
	// set it: the month picker's response already contains this card inside
	// the fragment being swapped, and marking it out-of-band there would
	// have htmx swap the card into an element it is mid-way through
	// replacing.
	OOB bool
}

// asOOB returns the card marked for out-of-band swapping, for the mutation
// handlers that send it alongside a transaction row.
func (c balanceCard) asOOB() balanceCard {
	c.OOB = true
	return c
}

// newBalanceCard computes a month's running balance and spending ratio from
// the three totals MonthlyTotals returns.
//
// The balance runs forward rather than resetting on the 1st: a month starts
// from whatever the ones before it left behind. Only the headline figure is
// cumulative -- SpentPct and RatioLabel below still measure the displayed
// month against its own income, because "spent 42% of this month's income"
// is the useful reading, and measuring a month's spending against a balance
// built up over years would report a steadily shrinking percentage that says
// nothing about how the month went.
//
// The arithmetic stays in int64 throughout: amounts are whole đồng, and
// going through float64 to compute a percentage would introduce rounding
// that the displayed number would eventually disagree with.
func newBalanceCard(expense, income, carriedOver int64, month time.Time, variant string) balanceCard {
	card := balanceCard{
		Expense:         expense,
		Income:          income,
		Remaining:       carriedOver + income - expense,
		CarriedOver:     carriedOver,
		ShowCarriedOver: carriedOver != 0,
		Month:           month,
		Variant:         variant,
		Empty:           carriedOver == 0 && expense == 0 && income == 0,
		Label:           balanceLabel(month, variant),
		RatioLabel:      "No income this month",
	}

	if income <= 0 {
		return card
	}

	pct := int(expense * 100 / income)
	switch {
	case expense > income:
		card.SpentPct = 100
		card.RatioLabel = "Over this month's income"
	default:
		card.SpentPct = pct
		card.LeftPct = 100 - pct
		card.ShowLeftPct = true
		card.RatioLabel = fmt.Sprintf("Spent %d%% of this month's income", pct)
	}
	return card
}

// balanceLabel names the number above the bar. Neither variant says "left"
// any more: the figure stopped being what remains of a month's income once
// it started carrying forward.
//
// The transactions page names the month ("Balance in August") because it can
// be browsing any month and the figure is that month's closing balance; the
// dashboard says it plain to match the wording of the Spent/Earned cards
// beside it.
func balanceLabel(month time.Time, variant string) string {
	if variant == "dashboard" {
		return "Balance"
	}
	return "Balance in " + monthLabelLower(month)
}

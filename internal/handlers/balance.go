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
	Expense, Income, Remaining int64
	Month                      time.Time
	Variant                    string

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

	// Empty marks a month with no transactions of either kind, where the
	// card still renders but its 0₫ is greyed to ink-zero.
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

// newBalanceCard computes a month's balance and spending ratio from the two
// totals MonthlyTotals returns.
//
// The arithmetic stays in int64 throughout: amounts are whole đồng, and
// going through float64 to compute a percentage would introduce rounding
// that the displayed number would eventually disagree with.
func newBalanceCard(expense, income int64, month time.Time, variant string) balanceCard {
	card := balanceCard{
		Expense:    expense,
		Income:     income,
		Remaining:  income - expense,
		Month:      month,
		Variant:    variant,
		Empty:      expense == 0 && income == 0,
		Label:      balanceLabel(month, variant),
		RatioLabel: "No income this month",
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

// balanceLabel names the number above the bar. The transactions page sits
// under a month picker that already names the month, so it repeats it
// ("Left in August"); the dashboard says "this month" to match the wording
// of the Spent/Earned cards beside it.
func balanceLabel(month time.Time, variant string) string {
	if variant == "dashboard" {
		return "Left this month"
	}
	return "Left in " + monthLabelLower(month)
}

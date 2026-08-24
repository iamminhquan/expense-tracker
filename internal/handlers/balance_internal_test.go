package handlers

import (
	"testing"
	"time"
)

func augustMonth() time.Time {
	return time.Date(2026, time.August, 1, 0, 0, 0, 0, vietnamLocation)
}

// The ratio bar and its caption are the whole point of the balance card, and
// every interesting case is a division that either can't be done (no income)
// or runs past the end of the bar (overspending). They are computed in Go so
// the template never has to branch on them.
func TestNewBalanceCardRatio(t *testing.T) {
	tests := []struct {
		name            string
		expense, income int64
		wantRemaining   int64
		wantSpentPct    int
		wantLeftPct     int
		wantShowLeftPct bool
		wantRatioLabel  string
		wantEmpty       bool
	}{
		{
			name:    "spent part of the month's income",
			expense: 5800000, income: 10000000,
			wantRemaining: 4200000, wantSpentPct: 58, wantLeftPct: 42,
			wantShowLeftPct: true, wantRatioLabel: "Spent 58% of this month's income",
		},
		{
			name:    "no income, expenses only",
			expense: 1200000, income: 0,
			wantRemaining: -1200000, wantSpentPct: 0, wantLeftPct: 0,
			wantShowLeftPct: false, wantRatioLabel: "No income this month",
		},
		{
			name:    "empty month",
			expense: 0, income: 0,
			wantRemaining: 0, wantSpentPct: 0, wantLeftPct: 0,
			wantShowLeftPct: false, wantRatioLabel: "No income this month",
			wantEmpty: true,
		},
		{
			name:    "spent more than the month's income",
			expense: 12000000, income: 10000000,
			wantRemaining: -2000000, wantSpentPct: 100, wantLeftPct: 0,
			wantShowLeftPct: false, wantRatioLabel: "Over this month's income",
		},
		{
			// Exactly break-even is not overspending: the bar is full, the
			// caption still reads as a percentage, and "0% left" is true.
			name:    "spent exactly the month's income",
			expense: 10000000, income: 10000000,
			wantRemaining: 0, wantSpentPct: 100, wantLeftPct: 0,
			wantShowLeftPct: true, wantRatioLabel: "Spent 100% of this month's income",
		},
		{
			// Integer division truncates rather than rounding, so a third of
			// the income reads 33%, never 33.33 and never 34.
			name:    "fraction truncates instead of rounding",
			expense: 1000000, income: 3000000,
			wantRemaining: 2000000, wantSpentPct: 33, wantLeftPct: 67,
			wantShowLeftPct: true, wantRatioLabel: "Spent 33% of this month's income",
		},
		{
			name:    "income only",
			expense: 0, income: 7700000,
			wantRemaining: 7700000, wantSpentPct: 0, wantLeftPct: 100,
			wantShowLeftPct: true, wantRatioLabel: "Spent 0% of this month's income",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newBalanceCard(tc.expense, tc.income, 0, augustMonth(), "transactions")
			if got.Remaining != tc.wantRemaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tc.wantRemaining)
			}
			if got.SpentPct != tc.wantSpentPct {
				t.Errorf("SpentPct = %d, want %d", got.SpentPct, tc.wantSpentPct)
			}
			if got.LeftPct != tc.wantLeftPct {
				t.Errorf("LeftPct = %d, want %d", got.LeftPct, tc.wantLeftPct)
			}
			if got.ShowLeftPct != tc.wantShowLeftPct {
				t.Errorf("ShowLeftPct = %v, want %v", got.ShowLeftPct, tc.wantShowLeftPct)
			}
			if got.RatioLabel != tc.wantRatioLabel {
				t.Errorf("RatioLabel = %q, want %q", got.RatioLabel, tc.wantRatioLabel)
			}
			if got.Empty != tc.wantEmpty {
				t.Errorf("Empty = %v, want %v", got.Empty, tc.wantEmpty)
			}
			if got.Expense != tc.expense || got.Income != tc.income {
				t.Errorf("Expense/Income = %d/%d, want %d/%d", got.Expense, got.Income, tc.expense, tc.income)
			}
		})
	}
}

// A negative balance is the case vndBalance exists for -- the card must show
// the minus sign rather than the bare magnitude vnd would give it. A positive
// one carries no sign at all: this is a standing amount, not a change.
func TestNewBalanceCardFormatsNegativeRemaining(t *testing.T) {
	card := newBalanceCard(1200000, 0, 0, augustMonth(), "transactions")
	if got := vndBalance(card.Remaining); got != "-1,200,000₫" {
		t.Errorf("vndBalance(Remaining) = %q, want %q", got, "-1,200,000₫")
	}
	positive := newBalanceCard(2300000, 10000000, 0, augustMonth(), "transactions")
	if got := vndBalance(positive.Remaining); got != "7,700,000₫" {
		t.Errorf("vndBalance(Remaining) = %q, want %q", got, "7,700,000₫")
	}
}

// The balance runs forward across months rather than resetting on the 1st:
// what a month ends with is what the next one starts from. Only the headline
// figure is cumulative -- the ratio bar and the Spent/Earned pair still
// describe the displayed month alone, which is what makes "spent 58% of this
// month's income" a true statement next to a balance built over years.
func TestNewBalanceCardCarriesTheBalanceForward(t *testing.T) {
	tests := []struct {
		name            string
		expense, income int64
		carriedOver     int64
		wantRemaining   int64
		wantSpentPct    int
		wantRatioLabel  string
		wantEmpty       bool
	}{
		{
			name:    "carried-in balance adds to the month's own net",
			expense: 4200000, income: 10000000, carriedOver: 6650000,
			wantRemaining: 12450000,
			wantSpentPct:  42, wantRatioLabel: "Spent 42% of this month's income",
		},
		{
			// The month the app was first used has nothing behind it.
			name:    "nothing carried in leaves the month's own net alone",
			expense: 4200000, income: 10000000, carriedOver: 0,
			wantRemaining: 5800000,
			wantSpentPct:  42, wantRatioLabel: "Spent 42% of this month's income",
		},
		{
			// The case that motivated the whole feature: a fresh month with
			// nothing entered yet still shows what last month left behind,
			// where it used to grey out a 0₫.
			name:    "a month with no transactions still shows what came before",
			expense: 0, income: 0, carriedOver: 6650000,
			wantRemaining: 6650000,
			wantSpentPct:  0, wantRatioLabel: "No income this month",
		},
		{
			// Empty is about having nothing to show, and a carried-in balance
			// is something to show. Only a user with no history at all sees
			// the greyed zero now.
			name:    "empty means no history at all, not just an empty month",
			expense: 0, income: 0, carriedOver: 0,
			wantRemaining: 0,
			wantSpentPct:  0, wantRatioLabel: "No income this month", wantEmpty: true,
		},
		{
			// Overspending in earlier months carries its debt forward too.
			name:    "a negative carry-in stays negative",
			expense: 500000, income: 0, carriedOver: -2000000,
			wantRemaining: -2500000,
			wantSpentPct:  0, wantRatioLabel: "No income this month",
		},
		{
			// A big enough carry-in keeps the balance positive through a month
			// that overspent its own income -- the bar still reports the
			// overspend, because the bar is a statement about the month.
			name:    "the bar reports the month's overspend even while the balance stays positive",
			expense: 12000000, income: 10000000, carriedOver: 30000000,
			wantRemaining: 28000000,
			wantSpentPct:  100, wantRatioLabel: "Over this month's income",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := newBalanceCard(tc.expense, tc.income, tc.carriedOver, augustMonth(), "transactions")
			if got.Remaining != tc.wantRemaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tc.wantRemaining)
			}
			if got.SpentPct != tc.wantSpentPct {
				t.Errorf("SpentPct = %d, want %d", got.SpentPct, tc.wantSpentPct)
			}
			if got.RatioLabel != tc.wantRatioLabel {
				t.Errorf("RatioLabel = %q, want %q", got.RatioLabel, tc.wantRatioLabel)
			}
			if got.Empty != tc.wantEmpty {
				t.Errorf("Empty = %v, want %v", got.Empty, tc.wantEmpty)
			}
			// The month's own two figures must survive untouched; the
			// carry-in is added to the headline, never folded into either.
			if got.Expense != tc.expense || got.Income != tc.income {
				t.Errorf("Expense/Income = %d/%d, want %d/%d", got.Expense, got.Income, tc.expense, tc.income)
			}
		})
	}
}

// The two pages name the same number differently: the transactions page is
// already headed by a month picker naming the month, so it repeats it; the
// dashboard says "this month" to match the wording of the Spent/Earned cards
// beside it.
func TestNewBalanceCardLabelPerVariant(t *testing.T) {
	if got := newBalanceCard(0, 0, 0, augustMonth(), "transactions"); got.Label != "Left in August" {
		t.Errorf("transactions Label = %q, want %q", got.Label, "Left in August")
	}
	if got := newBalanceCard(0, 0, 0, augustMonth(), "dashboard"); got.Label != "Left this month" {
		t.Errorf("dashboard Label = %q, want %q", got.Label, "Left this month")
	}
}

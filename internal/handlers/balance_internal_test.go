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
			got := newBalanceCard(tc.expense, tc.income, augustMonth(), "transactions")
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
// the minus sign rather than the bare magnitude vnd would give it.
func TestNewBalanceCardFormatsNegativeRemaining(t *testing.T) {
	card := newBalanceCard(1200000, 0, augustMonth(), "transactions")
	if got := vndBalance(card.Remaining); got != "-1,200,000₫" {
		t.Errorf("vndBalance(Remaining) = %q, want %q", got, "-1,200,000₫")
	}
	positive := newBalanceCard(2300000, 10000000, augustMonth(), "transactions")
	if got := vndBalance(positive.Remaining); got != "+7,700,000₫" {
		t.Errorf("vndBalance(Remaining) = %q, want %q", got, "+7,700,000₫")
	}
}

// The two pages name the same number differently: the transactions page is
// already headed by a month picker naming the month, the dashboard card says
// "this month" the way its sibling Spent/Earned cards do.
func TestNewBalanceCardLabelPerVariant(t *testing.T) {
	if got := newBalanceCard(0, 0, augustMonth(), "transactions"); got.Label != "Left in August" {
		t.Errorf("transactions Label = %q, want %q", got.Label, "Left in August")
	}
	if got := newBalanceCard(0, 0, augustMonth(), "dashboard"); got.Label != "Left this month" {
		t.Errorf("dashboard Label = %q, want %q", got.Label, "Left this month")
	}
}

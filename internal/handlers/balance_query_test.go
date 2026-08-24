package handlers_test

import (
	"context"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func day(t *testing.T, s string) pgtype.Date {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse date %q: %v", s, err)
	}
	return pgtype.Date{Time: d, Valid: true}
}

// carryoverFixture creates a user with three months of history behind
// August 2026 and returns their ID.
func carryoverFixture(t *testing.T, deps handlers.Deps, email string) int64 {
	t.Helper()
	ctx := context.Background()
	deps.DB.Exec(ctx, "DELETE FROM users WHERE email = $1", email)

	user, err := deps.Queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email: email, PasswordHash: "x", Name: "Carry Over", Username: usernameFor(email),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})
	insertCarryoverHistory(t, deps, user.ID)
	return user.ID
}

// insertCarryoverHistory gives a user June, July and August 2026. The months
// are deliberately mixed -- June ends up ahead, July overspends -- so a
// carry-in that simply summed income, or dropped the sign, would land on a
// different number than the right one.
//
//	June   +10,000,000 -3,000,000 = +7,000,000
//	July    +8,000,000 -9,350,000 = -1,350,000  -> August carries in 5,650,000
//	August +10,000,000 -4,200,000 = +5,800,000  -> August closes at 11,450,000
func insertCarryoverHistory(t *testing.T, deps handlers.Deps, userID int64) {
	t.Helper()
	ctx := context.Background()
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", userID)
	})

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories to exist: %v", err)
	}
	expenseID := firstCategoryOfType(t, categories, "expense").ID
	incomeID := firstCategoryOfType(t, categories, "income").ID

	rows := []struct {
		date   string
		amount int64
		typ    string
	}{
		{"2026-06-05", 10000000, "income"}, // June nets +7,000,000
		{"2026-06-18", 3000000, "expense"}, //
		{"2026-07-03", 8000000, "income"},  // July nets -1,350,000
		{"2026-07-22", 9350000, "expense"}, //
		{"2026-08-04", 10000000, "income"}, // August nets +5,800,000
		{"2026-08-19", 4200000, "expense"}, //
	}
	for _, r := range rows {
		categoryID := expenseID
		if r.typ == "income" {
			categoryID = incomeID
		}
		if _, err := deps.DB.Exec(ctx,
			"INSERT INTO transactions (user_id, category_id, amount, type, occurred_on) VALUES ($1,$2,$3,$4,$5)",
			userID, categoryID, r.amount, r.typ, r.date,
		); err != nil {
			t.Fatalf("insert transaction %v: %v", r, err)
		}
	}
}

// The balance runs forward across months, so every month's card needs to
// know what the months before it left behind. That number comes back from
// MonthlyTotals as a third column rather than from a query of its own,
// because all five places that build a balance card already run this one and
// a separate query would mean a second round trip at each of them.
func TestMonthlyTotalsReturnsTheBalanceCarriedIn(t *testing.T) {
	deps := newTestDeps(t)
	userID := carryoverFixture(t, deps, "carryover-query@example.com")

	totals := func(from, to string) sqlcgen.MonthlyTotalsRow {
		t.Helper()
		row, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
			UserID: userID, OccurredOn: day(t, from), OccurredOn_2: day(t, to),
		})
		if err != nil {
			t.Fatalf("monthly totals %s..%s: %v", from, to, err)
		}
		return row
	}

	t.Run("the first month of history carries nothing in", func(t *testing.T) {
		got := totals("2026-06-01", "2026-07-01")
		if got.CarriedOver != 0 {
			t.Errorf("CarriedOver = %d, want 0", got.CarriedOver)
		}
	})

	t.Run("a later month carries in the net of everything before it", func(t *testing.T) {
		got := totals("2026-08-01", "2026-09-01")
		// June's +7,000,000 and July's -1,350,000.
		if got.CarriedOver != 5650000 {
			t.Errorf("CarriedOver = %d, want 5650000", got.CarriedOver)
		}
		// The month's own two totals must stay scoped to the month.
		if got.TotalExpense != 4200000 || got.TotalIncome != 10000000 {
			t.Errorf("expense/income = %d/%d, want 4200000/10000000", got.TotalExpense, got.TotalIncome)
		}
	})

	t.Run("an untouched next month still carries the balance in", func(t *testing.T) {
		// The case that motivated the feature: nothing has been entered for
		// September, and the balance must not read 0.
		got := totals("2026-09-01", "2026-10-01")
		if got.CarriedOver != 11450000 {
			t.Errorf("CarriedOver = %d, want 11450000", got.CarriedOver)
		}
		if got.TotalExpense != 0 || got.TotalIncome != 0 {
			t.Errorf("expense/income = %d/%d, want 0/0", got.TotalExpense, got.TotalIncome)
		}
	})
}

// The carry-in reaches back over a user's whole history, which makes it the
// query most exposed to leaking another user's rows: unlike the month totals
// it is not fenced in by a date range on both sides.
func TestMonthlyTotalsCarriedInIgnoresOtherUsers(t *testing.T) {
	deps := newTestDeps(t)
	userID := carryoverFixture(t, deps, "carryover-mine@example.com")
	carryoverFixture(t, deps, "carryover-theirs@example.com")

	got, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: day(t, "2026-08-01"), OccurredOn_2: day(t, "2026-09-01"),
	})
	if err != nil {
		t.Fatalf("monthly totals: %v", err)
	}
	if got.CarriedOver != 5650000 {
		t.Errorf("CarriedOver = %d, want 5650000 (the other user's history must not count)", got.CarriedOver)
	}
}

// The property that makes the balance trustworthy: no đồng may be created or
// lost at a month boundary. What a month carries in, plus what it earned and
// spent, must be exactly what the next month carries in -- for every
// boundary, including the ones on either side of a month that overspent and
// the one into a month with nothing in it at all.
func TestCarriedInIsContinuousAcrossEveryMonthBoundary(t *testing.T) {
	deps := newTestDeps(t)
	userID := carryoverFixture(t, deps, "carryover-continuity@example.com")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	incomeID := firstCategoryOfType(t, categories, "income").ID
	extra := []struct {
		date   string
		amount int64
	}{
		// Payday on the 1st is the boundary this whole feature turns on: it
		// belongs to its own month, never to the carry-in. Without a row
		// sitting exactly on a month boundary the continuity check below
		// cannot tell "< the 1st" from "<= the 1st".
		{"2026-07-01", 1000000},
		// A far-future one too, to prove nothing falls into a gap between
		// "this month" and "everything before it" while it waits for its
		// month to arrive.
		{"2026-11-10", 2500000},
	}
	for _, e := range extra {
		if _, err := deps.DB.Exec(context.Background(),
			"INSERT INTO transactions (user_id, category_id, amount, type, occurred_on) VALUES ($1,$2,$3,$4,$5)",
			userID, incomeID, e.amount, "income", e.date,
		); err != nil {
			t.Fatalf("insert transaction %v: %v", e, err)
		}
	}

	// The payday on 1 July belongs to July, not to what July carried in.
	july, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: day(t, "2026-07-01"), OccurredOn_2: day(t, "2026-08-01"),
	})
	if err != nil {
		t.Fatalf("monthly totals July: %v", err)
	}
	if july.CarriedOver != 7000000 {
		t.Errorf("July carried in %d, want 7000000 -- a transaction dated the 1st must count as the month's own", july.CarriedOver)
	}
	if july.TotalIncome != 9000000 {
		t.Errorf("July income = %d, want 9000000 (8,000,000 plus the payday on the 1st)", july.TotalIncome)
	}

	months := []string{
		"2026-05-01", "2026-06-01", "2026-07-01", "2026-08-01",
		"2026-09-01", "2026-10-01", "2026-11-01", "2026-12-01",
	}
	for i := 0; i < len(months)-1; i++ {
		from, to := months[i], months[i+1]
		row, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
			UserID: userID, OccurredOn: day(t, from), OccurredOn_2: day(t, to),
		})
		if err != nil {
			t.Fatalf("monthly totals %s: %v", from, err)
		}
		next, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
			UserID: userID, OccurredOn: day(t, to), OccurredOn_2: day(t, months[min(i+2, len(months)-1)]),
		})
		if err != nil {
			t.Fatalf("monthly totals %s: %v", to, err)
		}
		closing := row.CarriedOver + row.TotalIncome - row.TotalExpense
		if closing != next.CarriedOver {
			t.Errorf("%s closed at %d but %s carried in %d -- %d đồng went missing at the boundary",
				from, closing, to, next.CarriedOver, closing-next.CarriedOver)
		}
	}

	// And the whole history nets out to the same figure the last boundary
	// reports, so nothing was double counted along the way either.
	final, err := deps.Queries.MonthlyTotals(context.Background(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: day(t, "2026-12-01"), OccurredOn_2: day(t, "2027-01-01"),
	})
	if err != nil {
		t.Fatalf("monthly totals December: %v", err)
	}
	// 7,000,000 + (-1,350,000 + 1,000,000) + 5,800,000 + 2,500,000.
	if final.CarriedOver != 14950000 {
		t.Errorf("history nets to %d, want 14950000", final.CarriedOver)
	}
}

package handlers_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// filterFixture is the handful of transactions every filter test below
// narrows down. They differ in note, type, category and amount so that each
// criterion has something it alone selects, and something it alone rejects.
type filterFixture struct {
	cookie           *http.Cookie
	userID           int64
	foodID, salaryID int64
	firstOfMonth     pgtype.Date
}

func seedFilterFixture(t *testing.T, deps handlers.Deps, router http.Handler, email string) filterFixture {
	t.Helper()
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	user, err := deps.Queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{Int64: user.ID, Valid: true})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	food := firstCategoryOfType(t, categories, "expense")
	salary := firstCategoryOfType(t, categories, "income")

	now := time.Now()
	day := pgtype.Date{Time: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), Valid: true}
	var otherExpense sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" && c.ID != food.ID {
			otherExpense = c
			break
		}
	}
	if otherExpense.ID == 0 {
		t.Fatal("expected a second expense category to filter against")
	}

	for _, txn := range []sqlcgen.CreateTransactionParams{
		{CategoryID: food.ID, Amount: 45000, Type: "expense", Description: "Morning coffee"},
		{CategoryID: food.ID, Amount: 350000, Type: "expense", Description: "Dinner with COFFEE after"},
		{CategoryID: otherExpense.ID, Amount: 20000, Type: "expense", Description: "Bus ticket"},
		{CategoryID: salary.ID, Amount: 9000000, Type: "income", Description: "August payslip"},
	} {
		txn.UserID = user.ID
		txn.OccurredOn = day
		if _, err := deps.Queries.CreateTransaction(context.Background(), txn); err != nil {
			t.Fatalf("seed %q: %v", txn.Description, err)
		}
	}

	return filterFixture{cookie: cookie, userID: user.ID, foodID: food.ID, salaryID: salary.ID, firstOfMonth: day}
}

// assertRows checks exactly which of the fixture's notes the rendered list
// shows. Naming both halves is the point: a filter that returns everything
// and a filter that returns nothing would each satisfy a one-sided check.
func assertRows(t *testing.T, body string, want, notWant []string) {
	t.Helper()
	for _, note := range want {
		if !strings.Contains(body, note) {
			t.Errorf("expected %q in the filtered list", note)
		}
	}
	for _, note := range notWant {
		if strings.Contains(body, note) {
			t.Errorf("expected %q to be filtered out", note)
		}
	}
}

func TestSearchMatchesPartOfANoteRegardlessOfCase(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-search@example.com")

	body := getTransactions(t, router, f.cookie, "?q=coffee")

	assertRows(t, body,
		[]string{"Morning coffee", "Dinner with COFFEE after"},
		[]string{"Bus ticket", "August payslip"})
}

func TestFilteringByTypeKeepsOnlyThatSide(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-type@example.com")

	income := getTransactions(t, router, f.cookie, "?type=income")
	assertRows(t, income, []string{"August payslip"}, []string{"Morning coffee", "Bus ticket"})

	expense := getTransactions(t, router, f.cookie, "?type=expense")
	assertRows(t, expense, []string{"Morning coffee", "Bus ticket"}, []string{"August payslip"})
}

func TestFilteringByCategoryKeepsOnlyItsTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-category@example.com")

	body := getTransactions(t, router, f.cookie, "?category="+strconv.FormatInt(f.foodID, 10))

	assertRows(t, body,
		[]string{"Morning coffee", "Dinner with COFFEE after"},
		[]string{"Bus ticket", "August payslip"})
}

func TestFilteringByAmountRangeKeepsWhatFallsInside(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-amount@example.com")

	both := getTransactions(t, router, f.cookie, "?min=30000&max=400000")
	assertRows(t, both,
		[]string{"Morning coffee", "Dinner with COFFEE after"},
		[]string{"Bus ticket", "August payslip"})

	lowerOnly := getTransactions(t, router, f.cookie, "?min=400000")
	assertRows(t, lowerOnly, []string{"August payslip"}, []string{"Morning coffee", "Bus ticket"})

	upperOnly := getTransactions(t, router, f.cookie, "?max=30000")
	assertRows(t, upperOnly, []string{"Bus ticket"}, []string{"Morning coffee", "August payslip"})
}

func TestFiltersCombineAsAnAnd(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-combined@example.com")

	body := getTransactions(t, router, f.cookie, "?q=coffee&type=expense&max=100000")

	assertRows(t, body,
		[]string{"Morning coffee"},
		[]string{"Dinner with COFFEE after", "Bus ticket", "August payslip"})
}

// A minimum above the maximum is a contradiction, not an error: the list
// simply comes back empty, the same way the query would answer any other
// range nothing falls into.
func TestAnImpossibleAmountRangeReturnsNothing(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-impossible@example.com")

	body := getTransactions(t, router, f.cookie, "?min=900000&max=1000")

	assertRows(t, body, nil, []string{"Morning coffee", "Bus ticket", "August payslip"})
}

func TestUnusableFilterValuesNarrowNothing(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-junk@example.com")

	body := getTransactions(t, router, f.cookie, "?min=abc&category=xyz&type=transfer&q=%20%20")

	assertRows(t, body,
		[]string{"Morning coffee", "Bus ticket", "August payslip"},
		nil)
}

// Filtering by a category that belongs to somebody else selects nothing
// rather than reaching into their rows: the filtered query is still fenced
// by user_id. The category has to be a personal one -- the defaults are
// shared, so both users legitimately have transactions in those.
func TestFilteringByAnotherUsersCategoryLeaksNothing(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	mine := seedFilterFixture(t, deps, router, "filter-mine@example.com")
	theirs := seedFilterFixture(t, deps, router, "filter-theirs@example.com")

	theirCategory, err := deps.Queries.CreateCategory(context.Background(), sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: theirs.userID, Valid: true},
		Name:   "Their private category", Type: "expense", Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create their category: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", theirCategory.ID)
	})
	if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
		UserID: theirs.userID, CategoryID: theirCategory.ID, Amount: 12000, Type: "expense",
		Description: "Their secret snack", OccurredOn: theirs.firstOfMonth,
	}); err != nil {
		t.Fatalf("create their transaction: %v", err)
	}

	body := getTransactions(t, router, mine.cookie, "?category="+strconv.FormatInt(theirCategory.ID, 10))

	assertRows(t, body, nil, []string{"Their secret snack", "Morning coffee", "Bus ticket"})
}

func TestTheCountChipReportsWhatTheFilterMatched(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-count@example.com")

	if body := getTransactions(t, router, f.cookie, ""); !strings.Contains(body, "4 transactions") {
		t.Error("expected the unfiltered chip to report all 4 transactions")
	}
	if body := getTransactions(t, router, f.cookie, "?q=coffee"); !strings.Contains(body, "2 transactions") {
		t.Error("expected the chip to report the 2 matches, not the whole month")
	}
}

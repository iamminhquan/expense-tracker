package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// crossMonthFixture spreads one user's transactions over three calendar
// months, with a note that appears in all three. A search that is still
// fenced to one month finds one of them; a search over the whole history
// finds all three, which is the difference every test below turns on.
type crossMonthFixture struct {
	cookie *http.Cookie
	userID int64
}

// monthsBack is the first of the month n months before the current one,
// built by stepping from the first of this month so no day-of-month can
// overflow a shorter one on the way.
func monthsBack(n int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -n, 0)
}

func seedCrossMonthFixture(t *testing.T, deps handlers.Deps, router http.Handler, email string) crossMonthFixture {
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

	on := func(monthsAgo, day int) pgtype.Date {
		return pgtype.Date{Time: monthsBack(monthsAgo).AddDate(0, 0, day-1), Valid: true}
	}
	for _, txn := range []sqlcgen.CreateTransactionParams{
		{Amount: 45000, Description: "Coffee this month", OccurredOn: on(0, 5)},
		{Amount: 50000, Description: "Coffee last month", OccurredOn: on(1, 12)},
		{Amount: 80000, Description: "Coffee two months ago", OccurredOn: on(2, 20)},
		{Amount: 12000, Description: "Bus ticket last month", OccurredOn: on(1, 3)},
	} {
		txn.UserID, txn.CategoryID, txn.Type = user.ID, food.ID, "expense"
		if _, err := deps.Queries.CreateTransaction(context.Background(), txn); err != nil {
			t.Fatalf("seed %q: %v", txn.Description, err)
		}
	}

	return crossMonthFixture{cookie: cookie, userID: user.ID}
}

// The gap this feature closes: the same search, run against one month and
// against the whole history, and only one of them finds what is not in the
// month the page happens to be on.
func TestSearchingAllMonthsFindsWhatTheMonthViewCannot(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-search@example.com")

	thisMonth := getTransactions(t, router, f.cookie, "?q=coffee")
	assertRows(t, thisMonth,
		[]string{"Coffee this month"},
		[]string{"Coffee last month", "Coffee two months ago"})

	everyMonth := getTransactions(t, router, f.cookie, "?month=all&q=coffee")
	assertRows(t, everyMonth,
		[]string{"Coffee this month", "Coffee last month", "Coffee two months ago"},
		[]string{"Bus ticket last month"})
}

func TestTheAllMonthsListHoldsEveryTransactionThereIs(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-all@example.com")

	body := getTransactions(t, router, f.cookie, "?month=all")

	assertRows(t, body, []string{
		"Coffee this month", "Coffee last month", "Coffee two months ago", "Bus ticket last month",
	}, nil)
	if !strings.Contains(body, "4 transactions") {
		t.Error("expected the count chip to count the whole history, not one month")
	}
}

// Ordering is still newest-first, which across months means the months
// themselves come out in order rather than interleaved.
func TestTheAllMonthsListRunsNewestMonthFirst(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-order@example.com")

	body := getTransactions(t, router, f.cookie, "?month=all")

	assertRowOrder(t, body, []string{
		"Coffee this month", "Coffee last month", "Bus ticket last month", "Coffee two months ago",
	})
}

// Sorting by amount over a whole history is the combination the month view
// could not express at all: the biggest spend of the year, not of August.
func TestSortingByAmountReachesAcrossEveryMonth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-sort@example.com")

	body := getTransactions(t, router, f.cookie, "?month=all&sort=amount_desc")

	assertRowOrder(t, body, []string{
		"Coffee two months ago", "Coffee last month", "Coffee this month", "Bus ticket last month",
	})
}

// The picker is the one control that decides scope, so this is where the new
// mode has to be reachable from.
func TestTheMonthPickerOffersEveryMonthAtOnce(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-picker@example.com")

	body := getTransactions(t, router, f.cookie, "")

	if !strings.Contains(body, "/transactions?month=all") {
		t.Error("expected the picker to offer an all-months entry")
	}
	if strings.Count(body, ">All months<") < 2 {
		t.Error("expected both the desktop and the mobile picker to offer it")
	}
}

// The trigger has to say which scope is on, or the list looks like a month
// that has inexplicably grown other months' rows.
func TestThePickerTriggerNamesTheAllMonthsScope(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-label@example.com")

	body := getTransactions(t, router, f.cookie, "?month=all")

	if !strings.Contains(body, "All months") {
		t.Fatal("expected the trigger to name the scope")
	}
	if strings.Contains(body, monthsBack(0).Format("January 2006")+" <span") {
		t.Error("expected the trigger to stop naming a single month while showing every month")
	}
}

// The dashboard's charts are built month by month, so the scope must not
// reach them -- neither offered in their picker nor honoured if typed.
func TestTheDashboardStaysMonthlyEvenWhenAskedForAllMonths(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-dashboard@example.com")

	req := httptest.NewRequest(http.MethodGet, "/dashboard?month=all", nil)
	req.AddCookie(f.cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /dashboard?month=all: expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	if strings.Contains(body, "/dashboard?month=all") {
		t.Error("expected the dashboard picker not to offer an all-months entry")
	}
	if !strings.Contains(body, monthsBack(0).Format("January 2006")) {
		t.Error("expected the dashboard to fall back to the current month")
	}
}

// The export follows the list, scope included -- and its filename has to
// stop claiming a month it no longer covers.
func TestTheExportFollowsTheAllMonthsScope(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-export@example.com")

	rec := getExport(t, router, f.cookie, "?month=all&q=coffee")

	notes := noteColumn(t, exportRecords(t, rec))
	want := []string{"Coffee this month", "Coffee last month", "Coffee two months ago"}
	if strings.Join(notes, "|") != strings.Join(want, "|") {
		t.Errorf("export notes = %q, want %q", notes, want)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "spend-all.csv") {
		t.Errorf("Content-Disposition = %q, want it to name spend-all.csv", got)
	}
}

// A fragment request has to push the scope, or a reload lands back on a
// single month with the search still in the box and fewer rows under it.
func TestAnAllMonthsFragmentPushesTheScopeIntoTheURL(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-push@example.com")

	rec := getTransactionsFragment(t, router, f.cookie, "?month=all&q=coffee&type=&category=&min=&max=")

	if got := rec.Header().Get("HX-Push-Url"); got != "/transactions?month=all&q=coffee" {
		t.Errorf("pushed URL = %q, want %q", got, "/transactions?month=all&q=coffee")
	}
}

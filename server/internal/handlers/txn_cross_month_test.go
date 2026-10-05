package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

// dateShort renders "02 Aug" because, as its own comment says, the year is
// implied by the month filter. Across a whole history nothing implies it, so
// the row has to name it -- and only then, or every row of a single month
// would repeat the same year for no one's benefit.
func TestTheDateNamesItsYearOnlyWhenMonthsAreMixed(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-year@example.com")

	year := monthsBack(1).Format("2006")
	twoAgo := monthsBack(2).Format("02 Jan 2006")

	everyMonth := getTransactions(t, router, f.cookie, "?month=all")
	if !strings.Contains(everyMonth, monthsBack(2).AddDate(0, 0, 19).Format("02 Jan 2006")) {
		t.Errorf("expected an all-months row to carry its year, e.g. %q", twoAgo)
	}

	thisMonth := getTransactions(t, router, f.cookie, "")
	if strings.Contains(thisMonth, monthsBack(0).AddDate(0, 0, 4).Format("02 Jan 2006")) {
		t.Errorf("expected a single month's rows to leave the year %q implied", year)
	}
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

// Both empty states are month-scoped sentences ("No transactions in august",
// "Nothing in august fits what you asked for"). Neither is true of a whole
// history, so each needs its own wording rather than a month name that isn't
// there.
func TestTheEmptyStatesStopTalkingAboutAMonthWhenShowingEveryMonth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-empty@example.com")

	filtered := getTransactions(t, router, f.cookie, "?month=all&q=nothingmatchesthis")
	if !strings.Contains(filtered, "No transactions match") {
		t.Error("expected the filtered empty state to still blame the filters")
	}
	if strings.Contains(filtered, "Nothing in "+monthsBack(0).Format("January")) {
		t.Error("expected the filtered empty state to stop naming a month")
	}

	// A history with nothing in it at all is the other empty state.
	if _, err := deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", f.userID); err != nil {
		t.Fatalf("clear transactions: %v", err)
	}
	bare := getTransactions(t, router, f.cookie, "?month=all")
	if strings.Contains(bare, "No transactions in "+monthsBack(0).Format("January")) {
		t.Error("expected the month's own empty state to stay out of an all-months list")
	}
	if !strings.Contains(bare, "No transactions yet") {
		t.Error("expected an empty history to say so")
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

// categoryOfTransaction reads a seeded row's category, for the update and
// create forms below, which have to name one the user actually owns.
func categoryOfTransaction(t *testing.T, deps handlers.Deps, id int64) int64 {
	t.Helper()
	var categoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE id = $1", id).Scan(&categoryID); err != nil {
		t.Fatalf("category of transaction %d: %v", id, err)
	}
	return categoryID
}

// The row's date is a map key on three of the four paths that render a row,
// and a template that reads a key a map does not carry prints nothing at all
// rather than failing -- so a path that forgot to supply it would ship rows
// with an empty date column, and every assertion about the rest of that row
// would still pass. This walks all four and insists each one dated its row.
func TestEveryPathThatRendersARowFillsInItsDate(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedCrossMonthFixture(t, deps, router, "cross-rowdate@example.com")
	id := idOfTransaction(t, deps, f.userID, "Coffee this month")
	categoryID := strconv.FormatInt(categoryOfTransaction(t, deps, id), 10)
	csrf := csrfTokenFor(t, router)

	// Every path is told, through HX-Current-URL, that the page it is
	// answering is showing all months -- so every row it renders owes its year.
	day := monthsBack(0).AddDate(0, 0, 4)
	want := day.Format("02 Jan 2006")

	send := func(name, method, target string, form url.Values) string {
		t.Helper()
		var req *http.Request
		if form == nil {
			req = httptest.NewRequest(method, target, nil)
		} else {
			req = httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Current-URL", "http://example.test/transactions?month=all")
		req.AddCookie(f.cookie)
		withCSRF(req, csrf)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", name, rec.Code)
		}
		return rec.Body.String()
	}

	rowID := strconv.FormatInt(id, 10)
	paths := map[string]string{
		"the list": getTransactions(t, router, f.cookie, "?month=all"),
		"a view":   send("view", http.MethodGet, "/transactions/"+rowID+"/view", nil),
		"an update": send("update", http.MethodPatch, "/transactions/"+rowID, url.Values{
			"category_id": {categoryID}, "amount": {"46000"},
			"description": {"Coffee this month"}, "occurred_on": {day.Format("2006-01-02")},
		}),
		"a create": send("create", http.MethodPost, "/transactions", url.Values{
			"category_id": {categoryID}, "amount": {"9000"}, "type": {"expense"},
			"description": {"Fresh row"}, "occurred_on": {day.Format("2006-01-02")},
		}),
	}
	for name, body := range paths {
		if !strings.Contains(body, want) {
			t.Errorf("%s rendered a row with no date on it, expected %q", name, want)
		}
	}
}

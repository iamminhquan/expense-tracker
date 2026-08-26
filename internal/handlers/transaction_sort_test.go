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

// sortFixture is four transactions whose date order, amount order and reverse
// amount order are three different sequences. That is what makes each
// assertion below mean something: on a fixture where two of the orders agreed,
// a sort that quietly did nothing would still pass.
type sortFixture struct {
	cookie   *http.Cookie
	userID   int64
	foodID   int64
	firstDay time.Time
}

func seedSortFixture(t *testing.T, deps handlers.Deps, router http.Handler, email string) sortFixture {
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

	now := time.Now()
	day := func(d int) pgtype.Date {
		return pgtype.Date{Time: time.Date(now.Year(), now.Month(), d, 0, 0, 0, 0, time.UTC), Valid: true}
	}
	for _, txn := range []sqlcgen.CreateTransactionParams{
		{Amount: 20000, Description: "Newest small", OccurredOn: day(12)},
		{Amount: 900000, Description: "Oldest large", OccurredOn: day(2)},
		{Amount: 500000, Description: "Middle big", OccurredOn: day(8)},
		{Amount: 5000, Description: "Recent tiny", OccurredOn: day(10)},
	} {
		txn.UserID, txn.CategoryID, txn.Type = user.ID, food.ID, "expense"
		if _, err := deps.Queries.CreateTransaction(context.Background(), txn); err != nil {
			t.Fatalf("seed %q: %v", txn.Description, err)
		}
	}

	return sortFixture{cookie: cookie, userID: user.ID, foodID: food.ID, firstDay: day(1).Time}
}

// assertRowOrder checks that the notes appear in the rendered list in exactly
// the sequence given, and that none of them is missing -- an order assertion
// on a list that lost a row would otherwise still hold.
func assertRowOrder(t *testing.T, body string, want []string) {
	t.Helper()
	at := make([]int, len(want))
	for i, note := range want {
		at[i] = strings.Index(body, ">"+note+"<")
		if at[i] < 0 {
			t.Fatalf("expected %q in the list", note)
		}
	}
	for i := 1; i < len(want); i++ {
		if at[i] < at[i-1] {
			t.Fatalf("expected %q before %q, got the other way round", want[i-1], want[i])
		}
	}
}

// The order the list has always had, and still falls back to.
func TestTheListDefaultsToNewestFirst(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-default@example.com")

	body := getTransactions(t, router, f.cookie, "")

	assertRowOrder(t, body, []string{"Newest small", "Recent tiny", "Middle big", "Oldest large"})
}

func TestSortingByAmountPutsTheBiggestSpendFirst(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-desc@example.com")

	body := getTransactions(t, router, f.cookie, "?sort=amount_desc")

	assertRowOrder(t, body, []string{"Oldest large", "Middle big", "Newest small", "Recent tiny"})
}

func TestSortingByAmountAscendingPutsTheSmallestFirst(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-asc@example.com")

	body := getTransactions(t, router, f.cookie, "?sort=amount_asc")

	assertRowOrder(t, body, []string{"Recent tiny", "Newest small", "Middle big", "Oldest large"})
}

// An order nobody offered falls back to the default rather than erroring, the
// way an unusable filter narrows nothing.
func TestAnUnknownSortOrderFallsBackToNewestFirst(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-junk@example.com")

	body := getTransactions(t, router, f.cookie, "?sort=amount%3B+DROP+TABLE")

	assertRowOrder(t, body, []string{"Newest small", "Recent tiny", "Middle big", "Oldest large"})
}

// Sorting narrows nothing, so the list still holds the whole month and the
// chip still counts all of it.
func TestSortingKeepsEveryRowInTheMonth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-count@example.com")

	body := getTransactions(t, router, f.cookie, "?sort=amount_desc")

	if !strings.Contains(body, "4 transactions") {
		t.Error("expected the chip to still report all 4 transactions")
	}
	if strings.Contains(body, "No transactions match") {
		t.Error("expected a sorted list not to claim it is filtered")
	}
}

// Sorting and filtering compose: the order applies to what the filters left.
func TestSortingAppliesToTheFilteredRowsOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-filtered@example.com")

	body := getTransactions(t, router, f.cookie, "?min=10000&sort=amount_desc")

	assertRowOrder(t, body, []string{"Oldest large", "Middle big", "Newest small"})
	if strings.Contains(body, "Recent tiny") {
		t.Error("expected the 5,000₫ row to stay filtered out")
	}
}

func TestTheSortControlComesBackShowingTheChosenOrder(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-control@example.com")

	body := getTransactions(t, router, f.cookie, "?sort=amount_desc")

	if !strings.Contains(body, `name="sort"`) {
		t.Fatal("expected the filter form to carry a sort control")
	}
	if !strings.Contains(body, `value="amount_desc" selected`) {
		t.Error("expected the chosen order to come back selected")
	}
}

// The pushed URL is what a reload or a bookmark lands on, so it has to keep
// the order the same way it keeps the filters.
func TestASortedFragmentPushesTheOrderIntoTheURL(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-push@example.com")

	rec := getTransactionsFragment(t, router, f.cookie, "?q=&type=&category=&min=&max=&sort=amount_asc")

	want := "/transactions?month=" + time.Now().Format("2006-01") + "&sort=amount_asc"
	if got := rec.Header().Get("HX-Push-Url"); got != want {
		t.Errorf("expected pushed URL %q, got %q", want, got)
	}
}

// A new row belongs at the top of a list in date order. In amount order its
// place depends on the amount just typed, so the create answers with the
// whole section rebuilt -- the same answer a create issued from a filtered
// list already gets.
func TestCreatingWhileSortedRebuildsTheListInsteadOfPrependingARow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-create@example.com")

	form := url.Values{
		"category_id": {strconv.FormatInt(f.foodID, 10)},
		"amount":      {"7000"},
		"type":        {"expense"},
		"description": {"Bought a notebook"},
		"occurred_on": {f.firstDay.Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.test/transactions?sort=amount_desc")
	req.AddCookie(f.cookie)
	withCSRF(req, csrfTokenFor(t, router))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="transactions-month-section"`) {
		t.Error("expected the whole section back rather than a single prepended row")
	}
	assertRowOrder(t, rec.Body.String(), []string{"Oldest large", "Middle big", "Newest small", "Bought a notebook", "Recent tiny"})
}

// The panel is collapsed behind the overflow menu, and the badge that says a
// list is narrowed counts filters only -- so a sorted list would otherwise
// carry no sign at all of why it is in the order it is in. Being sorted opens
// the panel for the same reason being filtered does: the control that
// produced the view has to be visible from it.
func TestASortedListOpensThePanelThatSortedIt(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-panel@example.com")

	panelClass := func(body string) string {
		t.Helper()
		at := strings.Index(body, "data-filter-panel ")
		if at < 0 {
			t.Fatal("no filter panel in the rendered list")
		}
		return body[at : at+120]
	}

	if !strings.Contains(panelClass(getTransactions(t, router, f.cookie, "")), "hidden") {
		t.Error("expected the panel to stay collapsed on a plain list")
	}
	if strings.Contains(panelClass(getTransactions(t, router, f.cookie, "?sort=amount_desc")), "hidden") {
		t.Error("expected a sorted list to show the control that sorted it")
	}
}

// The export is meant to be the list on screen, which includes the order it
// is in: a CSV that came back newest-first from a list sorted by amount would
// not be the thing the user was looking at.
func TestTheExportKeepsTheOrderTheListIsIn(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedSortFixture(t, deps, router, "sort-export@example.com")

	notes := noteColumn(t, exportRecords(t, getExport(t, router, f.cookie, "?sort=amount_desc")))

	want := []string{"Oldest large", "Middle big", "Newest small", "Recent tiny"}
	if strings.Join(notes, "|") != strings.Join(want, "|") {
		t.Errorf("export order = %q, want %q", notes, want)
	}
}

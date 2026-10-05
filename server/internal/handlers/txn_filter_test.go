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

// getTransactionsFragment issues the request htmx issues for the list, and
// hands back the response so a test can look at its headers as well as its
// body.
func getTransactionsFragment(t *testing.T, router http.Handler, cookie *http.Cookie, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/transactions"+query, nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /transactions%s: expected 200, got %d", query, rec.Code)
	}
	return rec
}

func TestTheFilterFormShowsWhatIsBeingFiltered(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-form@example.com")

	category := strconv.FormatInt(f.foodID, 10)
	body := getTransactions(t, router, f.cookie, "?q=coffee&type=expense&category="+category+"&min=1000&max=90000")

	for _, want := range []string{
		`name="q"`, `value="coffee"`,
		`name="type"`, `name="category"`, `name="min"`, `name="max"`,
		`value="1000"`, `value="90000"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected the filter form to carry %s", want)
		}
	}
	if !strings.Contains(body, `value="`+category+`" selected`) {
		t.Error("expected the filtered category to come back selected")
	}
	if !strings.Contains(body, `value="expense" selected`) {
		t.Error("expected the filtered type to come back selected")
	}
}

func TestTheFilterBadgeCountsWhatIsActive(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-badge@example.com")

	plain := getTransactions(t, router, f.cookie, "")
	if strings.Contains(plain, `id="filter-badge"><span`) {
		t.Error("expected no badge while nothing is filtered")
	}

	filtered := getTransactions(t, router, f.cookie, "?q=coffee&type=expense&min=1000")
	if !strings.Contains(filtered, ">3<") {
		t.Error("expected the badge to count search, type and the amount range as 3")
	}
}

// The two empty lists are different situations and say different things: a
// month with nothing in it invites you to add something, a filter that
// matched nothing invites you to widen it.
func TestAnEmptyFilterResultSaysSoRatherThanClaimingTheMonthIsEmpty(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-empty@example.com")

	body := getTransactions(t, router, f.cookie, "?q=nothingmatchesthis")

	if !strings.Contains(body, "No transactions match") {
		t.Error("expected the empty state to blame the filters")
	}
	if strings.Contains(body, "No transactions in") {
		t.Error("expected the month's own empty state to stay out of a filtered list")
	}
	if !strings.Contains(body, "Clear filters") {
		t.Error("expected a way out of a filter that matched nothing")
	}
}

// The filter controls submit their whole form, so the URL htmx would push
// carries an empty parameter for every blank control. The handler pushes a
// canonical one instead.
func TestAFilteredFragmentPushesACanonicalURL(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-push@example.com")

	rec := getTransactionsFragment(t, router, f.cookie, "?q=coffee&type=&category=&min=&max=")

	want := "/transactions?month=" + time.Now().Format("2006-01") + "&q=coffee"
	if got := rec.Header().Get("HX-Push-Url"); got != want {
		t.Errorf("expected pushed URL %q, got %q", want, got)
	}
}

// Switching month has to carry the filters along, or the list would silently
// widen under the user. Both pickers -- the desktop dropdown and the mobile
// one -- do it by including the filter form in their own request.
func TestTheMonthPickerCarriesTheFiltersAlong(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-carry@example.com")

	body := getTransactions(t, router, f.cookie, "?q=coffee")

	if strings.Count(body, `hx-include="#transaction-filters"`) < 2 {
		t.Error("expected both month pickers to include the filter form")
	}
}

// Turning a page has to carry them too, which the fixture above is too small
// to show: with only four transactions there is no pager to inspect.
func TestThePagerCarriesTheFiltersAlong(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := pagingUser(t, deps, router, "filter-pager@example.com", 30)

	body := getTransactions(t, router, cookie, "?q=Txn")

	if !strings.Contains(body, "Page 1 of 3") {
		t.Fatal("expected 30 matching rows to still be paged")
	}
	// Only the pager's own markup counts here: the month pickers elsewhere on
	// the page carry the same attribute, and would mask its absence.
	pagerHTML := body[strings.Index(body, `id="transactions-pager"`):]
	pagerHTML = pagerHTML[:strings.Index(pagerHTML, "</div>")]
	if !strings.Contains(pagerHTML, `hx-include="#transaction-filters"`) {
		t.Errorf("expected the pager to include the filter form, got:\n%s", pagerHTML)
	}
}

// countTransactionRows counts the rows a response actually rendered, by the
// wrapper id every one of them carries.
func countTransactionRows(body string) int {
	return strings.Count(body, `id="transaction-row-`)
}

// TestAFilteredPageHoldsAFullPageOfRows pins the half of filtered paging
// that TestThePagerCarriesTheFiltersAlong does not. That one proves the
// pager sends the filters back; this one proves they reach the list query's
// LIMIT/OFFSET window and not just the count that sizes the pager.
//
// The two failures it rules out both leave a plausible-looking page. If the
// filter reached only the count, a page would be ten rows drawn from the
// unfiltered month with a pager sized for eleven matches; if it reached only
// the list, every match would arrive on one page with no pager at all.
func TestAFilteredPageHoldsAFullPageOfRows(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	// Thirty rows, amounts 1000 through 30000, so min=20000 keeps eleven of
	// them: one full page and one row over.
	cookie := pagingUser(t, deps, router, "filter-page-size@example.com", 30)

	page1 := getTransactions(t, router, cookie, "?min=20000")
	if got := countTransactionRows(page1); got != 10 {
		t.Errorf("filtered page 1 rendered %d rows, want a full page of 10", got)
	}
	// The pager has to be sized by the filtered count as well. Sized by the
	// unfiltered one it would offer a third page holding nothing, which the
	// row counts below would not notice.
	if !strings.Contains(page1, "Page 1 of 2") {
		t.Error("expected the pager to be sized by the eleven matches, not by the whole month")
	}
	page2 := getTransactions(t, router, cookie, "?min=20000&page=2")
	if got := countTransactionRows(page2); got != 1 {
		t.Errorf("filtered page 2 rendered %d rows, want the single row left over", got)
	}

	// The eleventh match belongs on page 2 and nowhere else.
	if strings.Contains(page1, "Txn 20") {
		t.Error("expected the 11th match to be held back for page 2")
	}
	if !strings.Contains(page2, "Txn 20") {
		t.Error("expected the 11th match on page 2")
	}
	// And a row the filter excludes may not pad either page out to ten.
	if strings.Contains(page1, "Txn 19") || strings.Contains(page2, "Txn 19") {
		t.Error("expected a row below the minimum to be filtered out of every page")
	}
}

// idOfTransaction finds a seeded row by its note.
func idOfTransaction(t *testing.T, deps handlers.Deps, userID int64, note string) int64 {
	t.Helper()
	var id int64
	err := deps.DB.QueryRow(context.Background(),
		"SELECT id FROM transactions WHERE user_id = $1 AND description = $2", userID, note).Scan(&id)
	if err != nil {
		t.Fatalf("find %q: %v", note, err)
	}
	return id
}

// A transaction added while a filter is on may not belong in the list it was
// added from, so the create answers with the whole section rebuilt rather
// than prepending a row that does not match -- the same answer it already
// gives a create issued from page 2.
func TestCreatingWhileFilteredRebuildsTheListInsteadOfPrependingARow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-create@example.com")

	form := url.Values{
		"category_id": {strconv.FormatInt(f.foodID, 10)},
		"amount":      {"31000"},
		"type":        {"expense"},
		"description": {"Bought a notebook"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.test/transactions?q=coffee")
	req.AddCookie(f.cookie)
	withCSRF(req, csrfTokenFor(t, router))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="transactions-month-section"`) {
		t.Error("expected the whole filtered section back, not a single row")
	}
	if rec.Header().Get("HX-Retarget") != "#transactions-month-section" {
		t.Errorf("expected htmx to be retargeted at the section, got %q", rec.Header().Get("HX-Retarget"))
	}
	if strings.Contains(body, "Bought a notebook") {
		t.Error("expected a row that does not match the filter to stay out of the list")
	}
	if !strings.Contains(body, "Morning coffee") {
		t.Error("expected the filter to still be applied to the rebuilt list")
	}
	if want := "/transactions?month=" + time.Now().Format("2006-01") + "&q=coffee"; rec.Header().Get("HX-Push-Url") != want {
		t.Errorf("expected the pushed URL to keep the filter, got %q", rec.Header().Get("HX-Push-Url"))
	}
}

// Deleting a row refreshes the count chip from the originating page's URL,
// filters included -- otherwise the chip would jump to the whole month's
// figure the moment anything was deleted.
func TestDeletingWhileFilteredLeavesAFilteredCount(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-delete@example.com")

	id := idOfTransaction(t, deps, f.userID, "Bus ticket")
	req := httptest.NewRequest(http.MethodDelete, "/transactions/"+strconv.FormatInt(id, 10), nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", "http://example.test/transactions?q=coffee")
	req.AddCookie(f.cookie)
	withCSRF(req, csrfTokenFor(t, router))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "2 transactions") {
		t.Error("expected the count to stay with the 2 rows the filter matched")
	}
	if strings.Contains(body, "3 transactions") {
		t.Error("expected the count to ignore rows the filter excludes")
	}
}

// categoryBySlug picks a shared default category out of the user's list.
// Tests name one by slug rather than by position or by displayed name, the
// same rule the rest of the app follows.
func categoryBySlug(t *testing.T, categories []sqlcgen.Category, slug string) sqlcgen.Category {
	t.Helper()
	for _, c := range categories {
		if c.Slug.Valid && c.Slug.String == slug {
			return c
		}
	}
	t.Fatalf("expected a default category with slug %q", slug)
	return sqlcgen.Category{}
}

func categoriesOf(t *testing.T, deps handlers.Deps, userID int64) []sqlcgen.Category {
	t.Helper()
	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{Int64: userID, Valid: true})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	return categories
}

// The category is on screen in every row, so a search for one is a search a
// user will reasonably try -- and it used to come back empty, because the
// term was only ever matched against the note.
func TestSearchMatchesADefaultCategoryByItsDisplayedName(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-catsearch@example.com")

	transport := categoryBySlug(t, categoriesOf(t, deps, f.userID), "transport")
	if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
		UserID: f.userID, CategoryID: transport.ID, Amount: 35000, Type: "expense",
		Description: "Grab home", OccurredOn: f.firstOfMonth,
	}); err != nil {
		t.Fatalf("seed transport transaction: %v", err)
	}

	body := getTransactions(t, router, f.cookie, "?q=Transport")

	assertRows(t, body, []string{"Grab home"}, []string{"Morning coffee", "August payslip"})
}

// A category the user named themselves has no slug and shows the words they
// typed, so that is what the search has to match.
func TestSearchMatchesAPersonalCategoryByItsOwnName(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-personalsearch@example.com")

	mine, err := deps.Queries.CreateCategory(context.Background(), sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: f.userID, Valid: true},
		Name:   "Cà phê sáng", Type: "expense", Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE category_id = $1", mine.ID)
		deps.DB.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", mine.ID)
	})
	if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
		UserID: f.userID, CategoryID: mine.ID, Amount: 25000, Type: "expense",
		Description: "Thursday", OccurredOn: f.firstOfMonth,
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	body := getTransactions(t, router, f.cookie, "?q=c%C3%A0+ph%C3%AA")

	assertRows(t, body, []string{"Thursday"}, []string{"Morning coffee", "Bus ticket"})
}

// Matching the category must not quietly widen the search: a term that only
// looks like a category to the naked eye still has to match something real.
func TestSearchStillMatchesNothingForATermNobodyUses(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-nomatch@example.com")

	body := getTransactions(t, router, f.cookie, "?q=Groceries")

	assertRows(t, body, nil, []string{"Morning coffee", "Bus ticket", "August payslip"})
}

// The count chip and the pager are built from the count query, so it has to
// match the category the same way the list does or the two disagree.
func TestTheCountChipCountsCategoryMatchesToo(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "filter-catcount@example.com")

	transport := categoryBySlug(t, categoriesOf(t, deps, f.userID), "transport")
	if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
		UserID: f.userID, CategoryID: transport.ID, Amount: 35000, Type: "expense",
		Description: "Grab home", OccurredOn: f.firstOfMonth,
	}); err != nil {
		t.Fatalf("seed transport transaction: %v", err)
	}

	body := getTransactions(t, router, f.cookie, "?q=Transport")

	if !strings.Contains(body, "1 transaction") {
		t.Error("expected the chip to count the row the category matched")
	}
}

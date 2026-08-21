package handlers_test

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func firstCategoryOfType(t *testing.T, categories []sqlcgen.Category, typ string) sqlcgen.Category {
	t.Helper()
	for _, c := range categories {
		if c.Type == typ {
			return c
		}
	}
	t.Fatalf("expected a category of type %q", typ)
	return sqlcgen.Category{}
}

func TestTransactionCRUDAndIsolation(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookieA := loginAndGetCookie(t, router, deps, "txn-a@example.com", "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "txn-b@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories to exist: %v", err)
	}
	categoryID := firstCategoryOfType(t, categories, "expense").ID

	today := time.Now().Format("2006-01-02")
	form := url.Values{
		"category_id": {strconv.FormatInt(categoryID, 10)},
		"amount":      {"50000"},
		"type":        {"expense"},
		"description": {"Cà phê"},
		"occurred_on": {today},
	}
	tok := csrfTokenFor(t, router)
	createReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(cookieA)
	withCSRF(createReq, tok)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK && createRec.Code != http.StatusSeeOther {
		t.Fatalf("expected success creating transaction, got %d: %s", createRec.Code, createRec.Body.String())
	}

	// Register cleanup immediately after creation (before any assertion that
	// could fail the test) so this test always deletes the row it created.
	// Without this, the leftover row references a default category and later
	// FK-violates TestMigrationsApplyCleanly's m.Down() when the full suite
	// runs (000005's down migration deletes default categories, which the
	// leftover transaction still references).
	userA, err := deps.Queries.GetUserByEmail(context.Background(), "txn-a@example.com")
	if err != nil {
		t.Fatalf("get user A: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(context.Background(), "DELETE FROM transactions WHERE user_id = $1", userA.ID)
	})

	listReqA := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReqA.AddCookie(cookieA)
	listRecA := httptest.NewRecorder()
	router.ServeHTTP(listRecA, listReqA)
	if !strings.Contains(listRecA.Body.String(), "Cà phê") {
		t.Fatal("expected user A to see their own transaction")
	}

	listReqB := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReqB.AddCookie(cookieB)
	listRecB := httptest.NewRecorder()
	router.ServeHTTP(listRecB, listReqB)
	if strings.Contains(listRecB.Body.String(), "Cà phê") {
		t.Fatal("expected user B NOT to see user A's transaction")
	}

	// Cross-user delete attempt: user B tries to delete user A's transaction
	// by guessing an ID from A's list. Extract the id from the row's root
	// element (id="transaction-row-{id}") to make the isolation check
	// concrete rather than assumed.
	bodyA := listRecA.Body.String()
	idx := strings.Index(bodyA, `id="transaction-row-`)
	if idx == -1 {
		t.Fatal("expected a transaction row in user A's page")
	}
	rest := bodyA[idx+len(`id="transaction-row-`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx == -1 {
		t.Fatal("expected a closing quote after the transaction row id")
	}
	txnID := rest[:endIdx]

	tokDel := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/transactions/"+txnID, nil)
	deleteReq.AddCookie(cookieB)
	withCSRF(deleteReq, tokDel)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	// Regardless of status code, the transaction must still exist for user A
	// afterward -- user B must not be able to delete it.
	verifyReqA := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	verifyReqA.AddCookie(cookieA)
	verifyRecA := httptest.NewRecorder()
	router.ServeHTTP(verifyRecA, verifyReqA)
	if !strings.Contains(verifyRecA.Body.String(), "Cà phê") {
		t.Fatal("expected user A's transaction to survive user B's delete attempt")
	}
}

// TestCreateTransactionRejectsForeignCategory covers Finding 1 from the Task 9
// review: a forged category_id belonging to another user's private category
// must be rejected instead of silently attached (which would leak that
// category's name/color to the attacker via the joined transactions list).
func TestCreateTransactionRejectsForeignCategory(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()

	_ = loginAndGetCookie(t, router, deps, "txn-owner@example.com", "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "txn-attacker@example.com", "s3cret-pass")

	userA, err := deps.Queries.GetUserByEmail(ctx, "txn-owner@example.com")
	if err != nil {
		t.Fatalf("get user A: %v", err)
	}

	// User A creates a private category that user B has no rights to.
	privateCategory, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: userA.ID, Valid: true},
		Name:   "Bí mật của A",
		Type:   "expense",
		Color:  "#5B8DEF",
	})
	if err != nil {
		t.Fatalf("create private category: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{
			ID:     privateCategory.ID,
			UserID: pgtype.Int8{Int64: userA.ID, Valid: true},
		})
	})

	// User B forges a request attaching a transaction to A's private category.
	today := time.Now().Format("2006-01-02")
	form := url.Values{
		"category_id": {strconv.FormatInt(privateCategory.ID, 10)},
		"amount":      {"12345"},
		"type":        {"expense"},
		"description": {"Forged"},
		"occurred_on": {today},
	}
	tok := csrfTokenFor(t, router)
	forgeReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	forgeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	forgeReq.AddCookie(cookieB)
	withCSRF(forgeReq, tok)
	forgeRec := httptest.NewRecorder()
	router.ServeHTTP(forgeRec, forgeReq)

	if forgeRec.Code != http.StatusBadRequest && forgeRec.Code != http.StatusForbidden {
		t.Fatalf("expected 400 or 403 rejecting forged category_id, got %d: %s", forgeRec.Code, forgeRec.Body.String())
	}
	if strings.Contains(forgeRec.Body.String(), "Bí mật của A") {
		t.Fatal("response must not leak the foreign private category's name")
	}

	// Verify no transaction row was created for user B referencing A's category.
	userB, err := deps.Queries.GetUserByEmail(ctx, "txn-attacker@example.com")
	if err != nil {
		t.Fatalf("get user B: %v", err)
	}
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	txns, err := deps.Queries.ListTransactionsForMonth(ctx, sqlcgen.ListTransactionsForMonthParams{
		UserID:       userB.ID,
		OccurredOn:   pgtype.Date{Time: monthStart, Valid: true},
		OccurredOn_2: pgtype.Date{Time: monthStart.AddDate(0, 1, 0), Valid: true},
	})
	if err != nil {
		t.Fatalf("list user B transactions: %v", err)
	}
	// Register cleanup for every transaction found before asserting on any of
	// them, so a failing assertion here still leaves the DB as it found it
	// (an unfixed version of this handler would create exactly one such row).
	var leaked bool
	for _, txn := range txns {
		t.Cleanup(func(id int64) func() {
			return func() {
				deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: id, UserID: userB.ID})
			}
		}(txn.ID))
		if txn.CategoryID == privateCategory.ID {
			leaked = true
		}
	}
	if leaked {
		t.Fatal("expected no transaction to be created against the forged category")
	}
}

func TestCreateTransactionRejectsTypeCategoryMismatch(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-type-mismatch@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"income"}, // deliberately mismatched
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "does not match") {
		t.Fatalf("expected a type/category mismatch error, got: %s", rec.Body.String())
	}
	if got := rec.Header().Get("HX-Retarget"); got != "#quick-add-form-wrapper" {
		t.Fatalf("expected HX-Retarget: #quick-add-form-wrapper, got %q", got)
	}
}

// TestCreateTransactionConsecutiveDesktopValidationErrorsBothRetarget covers
// a regression found in Task 9 review: handleCreateTransaction's desktop
// error path retargets "#quick-add-form-wrapper" with an outerHTML swap, so
// whatever markup that id lives on must be part of the swapped-in fragment
// itself -- otherwise the first error swap discards the id (and its
// "hidden md:block" breakpoint-gating classes) from the DOM, and a second
// consecutive failed submission can no longer be retargeted at all.
func TestCreateTransactionConsecutiveDesktopValidationErrorsBothRetarget(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-desktop-double-error@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"10000"},
		"type":        {"income"}, // deliberately mismatched, twice
		"occurred_on": {time.Now().Format("2006-01-02")},
	}

	for _, label := range []string{"first", "second"} {
		tok := csrfTokenFor(t, router)
		req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(cookie)
		withCSRF(req, tok)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s submission: expected 200, got %d: %s", label, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("HX-Retarget"); got != "#quick-add-form-wrapper" {
			t.Fatalf("%s submission: expected HX-Retarget: #quick-add-form-wrapper, got %q", label, got)
		}
		if !strings.Contains(rec.Body.String(), "does not match") {
			t.Fatalf("%s submission: expected a type/category mismatch error, got: %s", label, rec.Body.String())
		}
		// The swapped-in fragment must itself carry the wrapper's id and
		// breakpoint-gating classes, since an outerHTML swap replaces the
		// entire previous element -- including whatever the id/classes were
		// attached to -- with exactly this response body.
		if !strings.Contains(rec.Body.String(), `id="quick-add-form-wrapper"`) {
			t.Fatalf("%s submission: expected the retargeted fragment to carry id=\"quick-add-form-wrapper\" itself, got: %s", label, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "hidden md:block") {
			t.Fatalf("%s submission: expected the retargeted fragment to carry the desktop-only breakpoint classes, got: %s", label, rec.Body.String())
		}
	}
}

func TestCreateTransactionRejectsLongDescription(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-long-desc@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"description": {strings.Repeat("a", 201)},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "200 characters") {
		t.Fatalf("expected a description-length error, got: %s", rec.Body.String())
	}
}

func TestCreateTransactionRejectsFarFutureDate(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-far-future@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().AddDate(0, 0, 30).Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "too far in the future") {
		t.Fatalf("expected a far-future-date error, got: %s", rec.Body.String())
	}
}

func TestTransactionsPageFiltersByMonthParam(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-filter@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-month-filter@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	pastMonth := time.Now().AddDate(0, -2, 0)
	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 7000, Type: "expense",
		Description: "Old month txn",
		OccurredOn:  pgtype.Date{Time: time.Date(pastMonth.Year(), pastMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	currentReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	currentReq.AddCookie(cookie)
	currentRec := httptest.NewRecorder()
	router.ServeHTTP(currentRec, currentReq)
	if strings.Contains(currentRec.Body.String(), "Old month txn") {
		t.Fatal("expected the current-month view to NOT include a transaction from two months ago")
	}

	monthParam := pastMonth.Format("2006-01")
	pastReq := httptest.NewRequest(http.MethodGet, "/transactions?month="+monthParam, nil)
	pastReq.AddCookie(cookie)
	pastRec := httptest.NewRecorder()
	router.ServeHTTP(pastRec, pastReq)
	if !strings.Contains(pastRec.Body.String(), "Old month txn") {
		t.Fatalf("expected ?month=%s to include the past-month transaction, got: %s", monthParam, pastRec.Body.String())
	}
}

func TestMonthDropdownReturnsFragmentOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-fragment@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?month="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("expected a fragment response with no <html> wrapper, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `id="transactions-month-section"`) {
		t.Fatalf("expected the transactions_month_section fragment, got: %s", rec.Body.String())
	}
}

// The sticky mobile header (title, month picker, + add button) lives inside
// transactions_month_section and keys its content off .ActiveNav. The
// month-dropdown's own hx-get re-renders exactly that fragment, so if the
// handler ever renders it without setting ActiveNav, switching months wipes
// the header from the page until the next full load.
func TestMonthDropdownFragmentKeepsTheMobileHeader(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-header@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?month="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `aria-label="Add transaction"`) {
		t.Errorf("month-switch fragment lost the sticky header's add button: %s", body)
	}
	if !strings.Contains(body, "Transactions") {
		t.Errorf("month-switch fragment lost the sticky header's title: %s", body)
	}
}

func TestCreateTransactionAcceptsNearFutureDate(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-near-future@example.com", "s3cret-pass")
	ctx := context.Background()

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-near-future@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().AddDate(0, 0, 3).Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "too far in the future") {
		t.Fatalf("expected a 3-day-future date (within the 7-day threshold) to be accepted, got error in body: %s", rec.Body.String())
	}
}

func TestCreateTransactionViaMobileFormTriggersHXTrigger(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-mobile-create@example.com", "s3cret-pass")
	ctx := context.Background()

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-mobile-create@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"15000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().Format("2006-01-02")},
		"ui_source":   {"mobile"},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Trigger"); got != "transaction-created" {
		t.Fatalf("expected HX-Trigger: transaction-created, got %q", got)
	}
}

func TestCreateTransactionValidationErrorRetargetsMobileForm(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-mobile-error@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"15000"},
		"type":        {"income"}, // mismatched on purpose
		"occurred_on": {time.Now().Format("2006-01-02")},
		"ui_source":   {"mobile"},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("HX-Retarget"); got != "#mobile-quick-add-form" {
		t.Fatalf("expected HX-Retarget: #mobile-quick-add-form, got %q", got)
	}
	if rec.Header().Get("HX-Trigger") != "" {
		t.Fatal("expected no HX-Trigger on a validation-error response")
	}
}

func TestUpdateTransactionAppliesEdit(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-update@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-update@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 10000, Type: "expense",
		Description: "before edit", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"25000"},
		"description": {"after edit"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transactions/%d", txn.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 updating a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "after edit") {
		t.Fatalf("expected updated description in response, got: %s", rec.Body.String())
	}

	updated, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("get updated transaction: %v", err)
	}
	if updated.Amount != 25000 {
		t.Fatalf("expected amount 25000, got %d", updated.Amount)
	}
}

// TestCreateTransactionOOBTotalsUseActiveMonthFromHXCurrentURL covers
// Finding 3 from the final review: the create handler used to always
// compute its OOB totals fragment from currentMonthRange() (today's month),
// even when the page the request came from was displaying a different
// month via ?month=. htmx sends that page's full URL, query string
// included, in the HX-Current-URL request header -- this asserts the
// handler now reads it and returns totals for the month actually being
// viewed, not always today's.
func TestCreateTransactionOOBTotalsUseActiveMonthFromHXCurrentURL(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-create-active-month@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-create-active-month@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	// A past month, distinct from today's month, that already has one
	// transaction in it.
	pastMonth := time.Now().AddDate(0, -3, 0)
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 40000, Type: "expense",
		Description: "existing past-month txn",
		OccurredOn:  pgtype.Date{Time: time.Date(pastMonth.Year(), pastMonth.Month(), 5, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatalf("seed existing past-month transaction: %v", err)
	}

	// Simulate the page currently being on that past month (via ?month=)
	// and creating a second transaction backdated into the same month.
	monthParam := pastMonth.Format("2006-01")
	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"25000"},
		"type":        {"expense"},
		"occurred_on": {time.Date(pastMonth.Year(), pastMonth.Month(), 12, 0, 0, 0, 0, time.UTC).Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Current-URL", "http://example.com/transactions?month="+monthParam)
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// The active (past) month's total is 40000+25000=65000 and count is 2.
	// A handler still using currentMonthRange() would report today's month
	// instead, which has no transactions for this fresh user (0₫, 0 count).
	if !strings.Contains(body, "65,000") {
		t.Fatalf("expected OOB totals to reflect the active past month's 65,000₫ total, got: %s", body)
	}
	if !strings.Contains(body, "2 transactions") {
		t.Fatalf("expected OOB totals count of 2 for the active past month, got: %s", body)
	}
}

// TestDeleteTransactionOOBTotalsUseActiveMonthFromHXCurrentURL covers the
// delete side of Finding 3: deleting a transaction while viewing a past
// month (via ?month=, recovered server-side from HX-Current-URL) must
// return OOB totals for that past month, not today's.
func TestDeleteTransactionOOBTotalsUseActiveMonthFromHXCurrentURL(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-delete-active-month@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-delete-active-month@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	pastMonth := time.Now().AddDate(0, -2, 0)
	pastTxn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 40000, Type: "expense",
		Description: "past month txn to delete",
		OccurredOn:  pgtype.Date{Time: time.Date(pastMonth.Year(), pastMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	if err != nil {
		t.Fatalf("create past-month transaction: %v", err)
	}
	// A much larger current-month transaction: if the handler wrongly uses
	// currentMonthRange(), its total will leak through as 99,000.
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 99000, Type: "expense",
		Description: "current month txn", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	}); err != nil {
		t.Fatalf("create current-month transaction: %v", err)
	}

	monthParam := pastMonth.Format("2006-01")
	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", pastTxn.ID), nil)
	req.Header.Set("HX-Current-URL", "http://example.com/transactions?month="+monthParam)
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "99,000") {
		t.Fatalf("expected OOB totals to reflect the active past month (now empty), not leak the current month's 99,000₫ total, got: %s", body)
	}
	if !strings.Contains(body, "0 transactions") {
		t.Fatalf("expected OOB totals count of 0 for the now-empty active past month, got: %s", body)
	}
}

// TestTransactionEmptyStateHidesOnCreateAndReappearsOnDelete covers Finding
// 4 from the final review: the "No transactions in ..." empty-state block
// is a sibling of #transaction-list, not inside it, so creating the first
// transaction (an OOB/afterbegin insert into the list) used to leave the
// empty-state message showing next to the new row until a manual reload. It
// also covers the reverse: deleting the last remaining transaction should
// bring the message back.
func TestTransactionEmptyStateHidesOnCreateAndReappearsOnDelete(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-empty-state@example.com", "s3cret-pass")
	ctx := context.Background()

	listReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if !strings.Contains(listRec.Body.String(), "No transactions in") {
		t.Fatal("expected the empty state message before any transaction exists")
	}

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	createReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(cookie)
	withCSRF(createReq, tok)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("expected 200 creating transaction, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `id="transactions-empty" hx-swap-oob="true"`) {
		t.Fatalf("expected the create response to include an OOB update targeting #transactions-empty, got: %s", createRec.Body.String())
	}
	if strings.Contains(createRec.Body.String(), "No transactions in") {
		t.Fatalf("expected the create response's empty-state OOB fragment to be cleared, got: %s", createRec.Body.String())
	}

	idx := strings.Index(createRec.Body.String(), `id="transaction-row-`)
	if idx == -1 {
		t.Fatal("expected a transaction row id in the create response")
	}
	rest := createRec.Body.String()[idx+len(`id="transaction-row-`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx == -1 {
		t.Fatal("expected a closing quote after the transaction row id")
	}
	txnID := rest[:endIdx]

	tokDel := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/transactions/"+txnID, nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tokDel)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting transaction, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), "No transactions in") {
		t.Fatalf("expected the delete response's OOB fragment to bring back the empty state, got: %s", deleteRec.Body.String())
	}
}

func TestDeleteTransactionRemovesRow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-delete-new@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-delete-new@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 9000, Type: "expense",
		Description: "to delete", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txn.ID), nil)
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID}); err == nil {
		t.Fatal("expected the transaction to no longer exist")
	}
}

// The mobile category chips highlight the selected one through a
// has-[:checked] variant. The first chip used to also carry the accent
// classes unconditionally, so once the user tapped another chip two of them
// looked selected at the same time.
func TestCategoryChipsHighlightOnlyTheCheckedChip(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "chips-test@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions/category-chips?type=expense", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from the chips fragment, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	if got := strings.Count(body, ` checked`); got != 1 {
		t.Fatalf("expected exactly one pre-checked chip, got %d: %s", got, body)
	}
	// Every accent class has to sit behind the has-[:checked] variant; a bare
	// one would light a chip up no matter which radio is actually selected.
	bare := strings.Count(body, "border-accent") - strings.Count(body, "has-[:checked]:border-accent")
	if bare != 0 {
		t.Fatalf("expected no unconditional border-accent on a chip, found %d: %s", bare, body)
	}
	bare = strings.Count(body, "bg-accent/[0.08]") - strings.Count(body, "has-[:checked]:bg-accent/[0.08]")
	if bare != 0 {
		t.Fatalf("expected no unconditional bg-accent on a chip, found %d: %s", bare, body)
	}
}

// The balance card is the reason the totals fragment exists: adding a
// transaction has to move the number and the ratio bar without a page
// reload, which means the response must carry the card marked out-of-band
// and recomputed, not just the new row.
func TestCreateTransactionUpdatesBalanceCardOutOfBand(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-balance-card@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-balance-card@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	income := firstCategoryOfType(t, categories, "income")
	expense := firstCategoryOfType(t, categories, "expense")
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	now := time.Now()
	thisMonth := func(day int) pgtype.Date {
		return pgtype.Date{Time: time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.UTC), Valid: true}
	}
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: income.ID, Amount: 10000000, Type: "income",
		Description: "salary", OccurredOn: thisMonth(1),
	}); err != nil {
		t.Fatalf("seed income: %v", err)
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expense.ID, 10)},
		"amount":      {"5800000"},
		"type":        {"expense"},
		"occurred_on": {thisMonth(2).Time.Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := html.UnescapeString(rec.Body.String())

	// Matched on the card's own tag, not anywhere in the body: the same
	// response also carries the count and empty-state fragments, whose
	// hx-swap-oob would otherwise satisfy a looser check.
	oobCard := regexp.MustCompile(`<div id="balance-card"[^>]*hx-swap-oob="true"`)
	if !oobCard.MatchString(body) {
		t.Fatalf("create response does not carry the balance card out-of-band:\n%s", body)
	}
	// 10,000,000 earned less 5,800,000 spent, and 58% of the income spent.
	if !strings.Contains(body, "+4,200,000₫") {
		t.Errorf("balance card does not show the recomputed balance of +4,200,000₫:\n%s", body)
	}
	if !strings.Contains(body, "Spent 58% of this month's income") {
		t.Errorf("balance card does not show the recomputed ratio caption:\n%s", body)
	}
	if !strings.Contains(body, "width: 58%") {
		t.Errorf("balance card's ratio bar was not resized to 58%%:\n%s", body)
	}
}

// Switching months re-renders the card with that month's figures. It rides
// inside the month fragment rather than swapping separately, so it must not
// mark itself out-of-band there -- htmx would lift it out of the fragment
// and swap it into the element that same fragment is replacing.
func TestMonthFragmentCarriesBalanceCardInlineNotOutOfBand(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-balance@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?month="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Count(body, `id="balance-card"`) != 1 {
		t.Fatalf(`expected exactly one balance card in the month fragment, got %d:\n%s`, strings.Count(body, `id="balance-card"`), body)
	}
	if regexp.MustCompile(`<div id="balance-card"[^>]*hx-swap-oob`).MatchString(body) {
		t.Error("the month fragment's balance card marks itself out-of-band")
	}
}

// One page, one balance. The figure used to appear in three places on the
// transactions page and disagreed with itself whenever one stopped updating.
func TestBalanceAppearsExactlyOncePerPage(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-one-balance@example.com", "s3cret-pass")

	for _, path := range []string{"/transactions", "/dashboard"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d", path, rec.Code)
		}
		if got := strings.Count(rec.Body.String(), `id="balance-card"`); got != 1 {
			t.Errorf(`GET %s rendered %d balance cards, want 1`, path, got)
		}
	}
}

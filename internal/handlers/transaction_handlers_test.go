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

func TestTransactionCRUDAndIsolation(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookieA := loginAndGetCookie(t, router, deps, "txn-a@example.com", "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "txn-b@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories to exist: %v", err)
	}
	categoryID := categories[0].ID

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
	// by guessing an ID from A's list. Extract the id from A's delete form
	// action to make the isolation check concrete rather than assumed.
	bodyA := listRecA.Body.String()
	idx := strings.Index(bodyA, `/transactions/`)
	if idx == -1 {
		t.Fatal("expected a transaction delete form in user A's page")
	}
	rest := bodyA[idx+len("/transactions/"):]
	endIdx := strings.Index(rest, "/delete")
	if endIdx == -1 {
		t.Fatal("expected /delete action in user A's page")
	}
	txnID := rest[:endIdx]

	deleteReq := httptest.NewRequest(http.MethodPost, "/transactions/"+txnID+"/delete", nil)
	deleteReq.AddCookie(cookieB)
	withCSRF(deleteReq, tok)
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
	if !strings.Contains(rec.Body.String(), "không khớp") {
		t.Fatalf("expected a type/category mismatch error, got: %s", rec.Body.String())
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
	if !strings.Contains(rec.Body.String(), "200 ký tự") {
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
	if !strings.Contains(rec.Body.String(), "tương lai") {
		t.Fatalf("expected a far-future-date error, got: %s", rec.Body.String())
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
	if strings.Contains(rec.Body.String(), "tương lai") {
		t.Fatalf("expected a 3-day-future date (within the 7-day threshold) to be accepted, got error in body: %s", rec.Body.String())
	}
}

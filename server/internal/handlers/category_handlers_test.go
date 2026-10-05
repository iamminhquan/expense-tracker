package handlers_test

import (
	"context"
	"fmt"
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

// usernameFor derives a deterministic, valid username from a test email's
// local part (matching the 000009 migration's ^[a-z][a-z0-9_]{2,19}$ CHECK),
// so every loginAndGetCookie caller gets a unique username without having to
// invent one per call site.
func usernameFor(email string) string {
	local := strings.ToLower(strings.SplitN(email, "@", 2)[0])
	local = strings.NewReplacer("-", "_", ".", "_").Replace(local)
	if len(local) > 20 {
		local = local[:20]
	}
	return local
}

func loginAndGetCookie(t *testing.T, router http.Handler, deps handlers.Deps, email, password string) *http.Cookie {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Cat Test"}, "email": {email}, "username": {usernameFor(email)}, "password": {password}, "password_confirm": {password}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after register")
	}
	return cookies[0]
}

func TestCreateAndListCategories(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-test@example.com", "s3cret-pass")

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Du lịch"}, "type": {"expense"}, "color": {"#D97757"}}
	req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusSeeOther {
		t.Fatalf("expected success status creating category, got %d: %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/categories", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing categories, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), "Du lịch") {
		t.Fatal("expected created category to appear in list page")
	}
	// The default categories render through their slug, so the page shows
	// the English label rather than whatever the name column holds.
	if !strings.Contains(listRec.Body.String(), "Food &amp; Drink") {
		t.Fatal("expected default category to appear in list page")
	}
}

func TestDeleteExpenseCategoryReassignsTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-reassign@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "cat-reassign@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	category, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: user.ID, Valid: true}, Name: "Sẽ bị gộp", Type: "expense", Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 5000, Type: "expense",
		Description: "reassign me", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", category.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting an expense category with transactions, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	moved, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("get reassigned transaction: %v", err)
	}
	other, err := deps.Queries.GetDefaultCategoryForReassignment(ctx)
	if err != nil {
		t.Fatalf("get Other default: %v", err)
	}
	if moved.CategoryID != other.ID {
		t.Fatalf("expected transaction to be reassigned to Other (id %d), got category_id %d", other.ID, moved.CategoryID)
	}
	if _, err := deps.Queries.GetCategoryForUser(ctx, sqlcgen.GetCategoryForUserParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}}); err == nil {
		t.Fatal("expected the deleted category to no longer exist")
	}
}

func TestDeleteIncomeCategoryWithTransactionsIsBlocked(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-income-block@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "cat-income-block@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	category, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: user.ID, Valid: true}, Name: "Thu riêng", Type: "income", Color: "#4FA871",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}})
	})

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 5000, Type: "income",
		Description: "blocks delete", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", category.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 blocking delete of an income category with transactions, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if _, err := deps.Queries.GetCategoryForUser(ctx, sqlcgen.GetCategoryForUserParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}}); err != nil {
		t.Fatal("expected the blocked category to still exist")
	}
}

func TestDeleteDefaultCategoryIsForbidden(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-default-delete@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	var defaultCategory sqlcgen.Category
	for _, c := range categories {
		if !c.UserID.Valid {
			defaultCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", defaultCategory.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting a default category, got %d", deleteRec.Code)
	}
}

func TestUpdateCategoryColorAppliesToDefaultCategory(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-color@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	target := categories[0]
	original := target.Color
	newColor := "#8B7BD8"
	if original == newColor {
		newColor = "#5B8DEF"
	}
	t.Cleanup(func() {
		deps.Queries.UpdateCategoryColor(context.Background(), sqlcgen.UpdateCategoryColorParams{ID: target.ID, UserID: pgtype.Int8{}, Color: original})
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{"color": {newColor}}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/categories/%d/color", target.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 changing a default category's color, got %d: %s", rec.Code, rec.Body.String())
	}
	updated, err := deps.Queries.GetCategoryForUser(context.Background(), sqlcgen.GetCategoryForUserParams{ID: target.ID, UserID: pgtype.Int8{}})
	if err != nil {
		t.Fatalf("get updated category: %v", err)
	}
	if updated.Color != newColor {
		t.Fatalf("expected color %q, got %q", newColor, updated.Color)
	}
}

// TestCategoryEmptyStateHidesOnCreateAndReappearsOnDelete covers Finding 4
// from the final review: the "You haven't created any categories yet" empty-state
// block is a sibling of the category list, not inside it, so creating the
// first custom category (an OOB insert into the list) used to leave the
// empty-state message showing alongside the new row until a manual reload.
// It also covers the reverse: deleting the last custom category should
// bring the message back.
func TestCategoryEmptyStateHidesOnCreateAndReappearsOnDelete(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-empty-state@example.com", "s3cret-pass")

	listReq := httptest.NewRequest(http.MethodGet, "/categories", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if !strings.Contains(listRec.Body.String(), "created any categories yet") {
		t.Fatal("expected the empty state message before any custom category exists")
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Việc riêng"}, "type": {"expense"}, "color": {"#D97757"}}
	createReq := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(cookie)
	withCSRF(createReq, tok)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("expected 200 creating category, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if !strings.Contains(createRec.Body.String(), `id="categories-empty" hx-swap-oob="true"`) {
		t.Fatalf("expected the create response to include an OOB update targeting #categories-empty, got: %s", createRec.Body.String())
	}
	if strings.Contains(createRec.Body.String(), "created any categories yet") {
		t.Fatalf("expected the create response's empty-state OOB fragment to be cleared, got: %s", createRec.Body.String())
	}

	idx := strings.Index(createRec.Body.String(), `id="category-row-`)
	if idx == -1 {
		t.Fatal("expected a category row id in the create response")
	}
	rest := createRec.Body.String()[idx+len(`id="category-row-`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx == -1 {
		t.Fatal("expected a closing quote after the category row id")
	}
	createdID, err := strconv.ParseInt(rest[:endIdx], 10, 64)
	if err != nil {
		t.Fatalf("parse created category id: %v", err)
	}

	tokDel := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", createdID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tokDel)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting category, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), "created any categories yet") {
		t.Fatalf("expected the delete response's OOB fragment to bring back the empty state, got: %s", deleteRec.Body.String())
	}
}

func TestRenameCategoryRejectsDefault(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-rename-default@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	target := categories[0]

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Hacked Name"}}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/categories/%d/name", target.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 renaming a default category, got %d", rec.Code)
	}
}

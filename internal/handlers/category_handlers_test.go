package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func loginAndGetCookie(t *testing.T, router http.Handler, deps handlers.Deps, email, password string) *http.Cookie {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Cat Test"}, "email": {email}, "password": {password}}
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
	form := url.Values{"name": {"Du lịch"}, "type": {"expense"}, "color": {"#111111"}}
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
	if !strings.Contains(listRec.Body.String(), "Ăn uống") {
		t.Fatal("expected default category to appear in list page")
	}
}

// TestDeleteCategoryInUseShowsFriendlyError covers Finding 7 from the final
// whole-branch review: transactions.category_id has no ON DELETE clause
// (RESTRICT by default), so deleting a category that still has transactions
// referencing it raises a foreign-key violation. Before the fix this
// surfaced as a bare 500; it should instead re-render the categories page
// with a friendly Vietnamese message via the existing {{.Error}} slot.
func TestDeleteCategoryInUseShowsFriendlyError(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-inuse@example.com", "s3cret-pass")

	ctx := context.Background()
	user, err := deps.Queries.GetUserByEmail(ctx, "cat-inuse@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	category, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: user.ID, Valid: true},
		Name:   "Sẽ bị khóa",
		Type:   "expense",
		Color:  "#654321",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID:      user.ID,
		CategoryID:  category.ID,
		Amount:      1000,
		Type:        "expense",
		Description: "blocks delete",
		OccurredOn:  pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
		deps.Queries.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}})
	})

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/categories/%d/delete", category.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering categories page with a friendly error, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), "Không thể xóa danh mục") {
		t.Fatalf("expected friendly FK-violation error message in body, got: %s", deleteRec.Body.String())
	}
	// The category and its blocking transaction must both still exist.
	if !strings.Contains(deleteRec.Body.String(), "Sẽ bị khóa") {
		t.Fatal("expected the in-use category to remain listed after the failed delete")
	}
}

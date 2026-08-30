package handlers_test

import (
	"context"
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

func TestDeleteAccountRemovesTheAccountAndItsOwnData(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "delete-me@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	category := personalCategory(t, deps, user.ID, "Delete Me Travel")
	seedTransactions(t, deps, user.ID, category.ID, 3)

	rec := postSettings(t, router, cookie, "/settings/delete", url.Values{
		"current_password": {"s3cret-pass"},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /settings/delete = %d, want %d: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if got, want := rec.Header().Get("Location"), "/login?deleted=1"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
	if _, err := deps.Queries.GetUserByEmail(ctx, email); err == nil {
		t.Error("GetUserByEmail(deleted address) returned a user, want an error")
	}
	if n := countRows(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("transactions left behind = %d, want 0", n)
	}
	if n := countRows(t, deps, "SELECT count(*) FROM categories WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("personal categories left behind = %d, want 0", n)
	}
	if n := countRows(t, deps, "SELECT count(*) FROM sessions WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("sessions left behind = %d, want 0", n)
	}
}

// personalCategory creates one category owned by userID, i.e. with a real
// user_id rather than the NULL the 9 shared defaults carry.
func personalCategory(t *testing.T, deps handlers.Deps, userID int64, name string) sqlcgen.Category {
	t.Helper()
	category, err := deps.Queries.CreateCategory(context.Background(), sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: userID, Valid: true}, Name: name, Type: "expense", Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}
	return category
}

func countRows(t *testing.T, deps handlers.Deps, query string, args ...any) int64 {
	t.Helper()
	var n int64
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := deps.DB.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

// TestSettingsPageOffersAnExportAboveTheDeleteForm pins the one piece of the
// danger zone that is not decoration: the CSV link sits above the delete
// form, because the data is what someone regrets losing and the empty
// account is not.
func TestSettingsPageOffersAnExportAboveTheDeleteForm(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "danger-zone@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /settings = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	export := strings.Index(body, "/transactions/export?month=all")
	form := strings.Index(body, `action="/settings/delete"`)
	switch {
	case export < 0:
		t.Fatal("settings page has no all-time CSV export link, want one in the danger zone")
	case form < 0:
		t.Fatal(`settings page has no form posting to /settings/delete`)
	case export > form:
		t.Error("the CSV export link renders below the delete form, want it above")
	}
}

// TestLoginPageConfirmsAFinishedDeletionOnlyWhenMarked covers the one screen
// a deleted account can still see. Without a line here the deletion looks
// like a logout, and there is no account left to go and check.
func TestLoginPageConfirmsAFinishedDeletionOnlyWhenMarked(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	const notice = "Your account has been deleted."
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"/login?deleted=1", true},
		{"/login", false},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if got := strings.Contains(rec.Body.String(), notice); got != tc.want {
			t.Errorf("GET %s shows %q = %v, want %v", tc.path, notice, got, tc.want)
		}
	}
}

func TestDeleteAccountRejectsAWrongPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "keep-me@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	rec := postSettings(t, router, cookie, "/settings/delete", url.Values{
		"current_password": {"not-the-password"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /settings/delete with a wrong password = %d, want %d", rec.Code, http.StatusOK)
	}
	if want := "That password is not correct."; !strings.Contains(rec.Body.String(), want) {
		t.Errorf("re-rendered settings page does not contain %q", want)
	}
	if _, err := deps.Queries.GetUserByEmail(ctx, email); err != nil {
		t.Errorf("GetUserByEmail(%q) = %v, want the account still there", email, err)
	}
}

// TestDeleteAccountLeavesSharedDefaultsAndOtherAccountsAlone guards the one
// predicate that could quietly take the whole install down with one account:
// the 9 shared default categories carry a NULL user_id, and every other
// account's transactions point at them.
func TestDeleteAccountLeavesSharedDefaultsAndOtherAccountsAlone(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()

	goingCookie := loginAndGetCookie(t, router, deps, "going@example.com", "s3cret-pass")
	going, err := deps.Queries.GetUserByEmail(ctx, "going@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	const staying = "staying@example.com"
	loginAndGetCookie(t, router, deps, staying, "s3cret-pass")
	stayer, err := deps.Queries.GetUserByEmail(ctx, staying)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", stayer.ID)
	})

	// Both accounts spend against the same shared default, which is what a
	// per-user delete has to leave standing.
	shared := sharedDefaultCategory(t, deps, going.ID)
	seedTransactions(t, deps, going.ID, shared.ID, 2)
	seedTransactions(t, deps, stayer.ID, shared.ID, 2)
	stayerOwn := personalCategory(t, deps, stayer.ID, "Staying Travel")
	defaultsBefore := countRows(t, deps, "SELECT count(*) FROM categories WHERE user_id IS NULL")

	rec := postSettings(t, router, goingCookie, "/settings/delete", url.Values{
		"current_password": {"s3cret-pass"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /settings/delete = %d, want %d: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}

	if got := countRows(t, deps, "SELECT count(*) FROM categories WHERE user_id IS NULL"); got != defaultsBefore {
		t.Errorf("shared default categories = %d, want %d", got, defaultsBefore)
	}
	if _, err := deps.Queries.GetUserByEmail(ctx, staying); err != nil {
		t.Errorf("GetUserByEmail(%q) = %v, want the other account untouched", staying, err)
	}
	if got := countRows(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", stayer.ID); got != 2 {
		t.Errorf("the other account's transactions = %d, want 2", got)
	}
	if got := countRows(t, deps, "SELECT count(*) FROM categories WHERE id = $1", stayerOwn.ID); got != 1 {
		t.Errorf("the other account's own category rows = %d, want 1", got)
	}
}

// sharedDefaultCategory returns one of the categories seeded by migration,
// i.e. one with a NULL user_id, as the account sees it in its own list.
func sharedDefaultCategory(t *testing.T, deps handlers.Deps, userID int64) sqlcgen.Category {
	t.Helper()
	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{Int64: userID, Valid: true})
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	for _, c := range categories {
		if !c.UserID.Valid && c.Type == "expense" {
			return sqlcgen.Category{ID: c.ID, Name: c.Name, Type: c.Type, Color: c.Color, UserID: c.UserID}
		}
	}
	t.Fatal("no shared default expense category found")
	return sqlcgen.Category{}
}

// TestDeleteAccountRemovesStoredBankEmails guards that a deleted account's
// bank_emails rows are gone too, one hop further out than the categories and
// sessions covered above. Ordering is not what makes this safe:
// transactions.bank_email_id is ON DELETE SET NULL, so nothing blocks the
// account delete either way -- this just confirms the rows are actually
// removed rather than left behind as orphans.
func TestDeleteAccountRemovesStoredBankEmails(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "delete-bank-email@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	token := enableInbox(t, deps, user.ID)
	body := inboxPayload("no-reply@mbbank.com.vn", token+"@in.example.site", "s", "<del@mail>", "x")
	if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
		t.Fatalf("seed email: POST /inbox = %d, want 200", rec.Code)
	}

	rec := postSettings(t, router, cookie, "/settings/delete", url.Values{
		"current_password": {"s3cret-pass"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /settings/delete = %d, want %d: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}

	n, err := deps.Queries.CountBankEmailsForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountBankEmailsForUser() error = %v", err)
	}
	if n != 0 {
		t.Errorf("bank_emails after account delete = %d, want 0", n)
	}
}

// TestDeletedEmailCanRegisterAgainAsAFreshAccount is the deliberate answer to
// "should the address be reserved". It is not: $pend shares no data between
// accounts, so the new one starts empty, and holding the address back would
// mean keeping the one piece of personal data the owner just asked to be rid
// of.
func TestDeletedEmailCanRegisterAgainAsAFreshAccount(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "reuse-me@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")

	first, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	seedTransactions(t, deps, first.ID, sharedDefaultCategory(t, deps, first.ID).ID, 2)

	rec := postSettings(t, router, cookie, "/settings/delete", url.Values{"current_password": {"s3cret-pass"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /settings/delete = %d, want %d: %s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}

	registerUser(t, router, csrfTokenFor(t, router), email, usernameFor(email), "n3w-password")
	second, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("registering %q again: %v", email, err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM users WHERE id = $1", second.ID)
	})
	if second.ID == first.ID {
		t.Fatalf("the re-registered account reuses id %d, want a new row", second.ID)
	}
	if n := countRows(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", second.ID); n != 0 {
		t.Errorf("the new account starts with %d transactions, want 0", n)
	}
}

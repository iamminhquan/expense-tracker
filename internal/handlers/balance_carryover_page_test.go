package handlers_test

import (
	"context"
	"html"
	"net/http"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// carryoverUser registers a user through the real register flow (so the test
// has a session cookie) and gives them the shared June/July/August history.
func carryoverUser(t *testing.T, router http.Handler, deps handlers.Deps, email string) *http.Cookie {
	t.Helper()
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	insertCarryoverHistory(t, deps, user.ID)
	return cookie
}

// carryoverPage fetches a page and decodes the HTML entities html/template
// emits, so assertions can be written the way the amounts read on screen.
func carryoverPage(t *testing.T, router http.Handler, cookie *http.Cookie, target string) string {
	t.Helper()
	return html.UnescapeString(getPage(t, router, cookie, target))
}

// The nav header widget is the only place a balance is shown -- the page-body
// balance card was retired once the header carried the same number -- so this
// is where carrying forward has to be visible.
//
// The fixture's three months all sit in the past, which makes the assertion
// stable whatever month the suite runs in: while August 2026 is the current
// month the header adds 5,650,000 carried in to August's own +5,800,000, and
// from September on it carries in the whole 11,450,000 against an empty
// month. Either way the balance is the same 11,450,000 -- which is the
// property being tested.
func TestHeaderBalanceCarriesForwardInsteadOfResetting(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := carryoverUser(t, router, deps, "carryover-header@example.com")

	body := carryoverPage(t, router, cookie, "/transactions")

	if !strings.Contains(body, "11,450,000₫") {
		t.Errorf("header did not show the balance carried forward from earlier months")
	}
	// The balance is a standing amount, not a change, so no leading plus.
	if strings.Contains(body, "+11,450,000₫") {
		t.Errorf("balance must not be rendered with a leading +")
	}
}

// The popover under the widget has to say what the number now means: a
// running balance, and how much of it was already there when the month
// started. Without that line the figure reads as this month's earnings.
func TestHeaderBalancePopoverNamesTheBalanceAndWhatItCarriedIn(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := carryoverUser(t, router, deps, "carryover-popover@example.com")

	body := carryoverPage(t, router, cookie, "/dashboard")

	if strings.Contains(body, "LEFT THIS MONTH") {
		t.Errorf("the popover still calls the figure what is left of the month")
	}
	if !strings.Contains(body, "BALANCE") {
		t.Errorf("expected the popover to be headed BALANCE")
	}
	if !strings.Contains(body, "carried over") {
		t.Errorf("expected the popover to name the part carried in from earlier months")
	}
	// The ratio caption still measures the month against its own income.
	if !strings.Contains(body, "of this month's income") && !strings.Contains(body, "No income this month") {
		t.Errorf("expected the monthly ratio caption to survive")
	}
}

// A user with no history at all still sees the greyed zero: carrying forward
// only changes what an empty month means when something came before it.
func TestHeaderBalanceStaysEmptyForAUserWithNoHistory(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "carryover-nohistory@example.com", "s3cret-pass")

	body := carryoverPage(t, router, cookie, "/dashboard")

	if !strings.Contains(body, "0₫") {
		t.Errorf("expected a user with no transactions to still see 0₫")
	}
	if strings.Contains(body, "carried over") {
		t.Errorf("a user with no history has nothing carried in to announce")
	}
}

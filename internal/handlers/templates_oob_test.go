package handlers_test

import (
	"os"
	"regexp"
	"testing"
)

// An hx-swap-oob fragment replaces the element the page rendered, so it has to
// carry the same responsive visibility classes. #totals-summary is the inline
// desktop-only totals row: the mobile viewport shows #mobile-stat-cards
// instead, and dropping `hidden md:` from the OOB copy made both show up at
// once after adding a transaction.
func TestTotalsSummaryOOBKeepsPageVisibilityClasses(t *testing.T) {
	classOf := func(t *testing.T, path, id string) string {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		re := regexp.MustCompile(`(?s)<div id="` + id + `"[^>]*?class="([^"]*)"`)
		m := re.FindSubmatch(body)
		if m == nil {
			t.Fatalf("no <div id=%q ... class=...> found in %s", id, path)
		}
		return string(m[1])
	}

	page := classOf(t, "../web/templates/transactions.html", "totals-summary")
	oob := classOf(t, "../web/templates/transaction_row.html", "totals-summary")

	if page != oob {
		t.Fatalf("#totals-summary classes drifted between the page and its OOB fragment:\n  page: %q\n  oob:  %q", page, oob)
	}
}

package handlers_test

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"
)

// getExport issues the download the Export link points at and returns the
// recorder, so a test can assert on the headers as well as the body.
func getExport(t *testing.T, router http.Handler, cookie *http.Cookie, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/transactions/export"+query, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /transactions/export%s: expected 200, got %d", query, rec.Code)
	}
	return rec
}

// exportRecords parses the response as CSV, stripping the UTF-8 BOM the
// export writes for Excel's benefit. Parsing rather than substring-matching
// is what catches a note containing a comma or a quote being written without
// escaping, which is the failure a Contains check would sail past.
func exportRecords(t *testing.T, rec *httptest.ResponseRecorder) [][]string {
	t.Helper()
	body := strings.TrimPrefix(rec.Body.String(), "\ufeff")
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatalf("parse export as CSV: %v", err)
	}
	return records
}

// noteColumn pulls the Note out of every data row, which is what the fixture
// tells its transactions apart by.
func noteColumn(t *testing.T, records [][]string) []string {
	t.Helper()
	var notes []string
	for _, row := range records[1:] {
		if len(row) != 5 {
			t.Fatalf("expected 5 columns per row, got %d: %q", len(row), row)
		}
		notes = append(notes, row[4])
	}
	return notes
}

func TestExportWritesOneRowPerTransactionUnderAMachineReadableHeader(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-columns@example.com")

	records := exportRecords(t, getExport(t, router, f.cookie, ""))

	wantHeader := []string{"Date", "Type", "Category", "Amount", "Note"}
	if len(records) == 0 || !equalRow(records[0], wantHeader) {
		t.Fatalf("first row = %q, want header %q", records[0], wantHeader)
	}
	if got := len(records) - 1; got != 4 {
		t.Fatalf("exported %d transactions, want the fixture's 4", got)
	}
}

func equalRow(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestExportNarrowsToTheSameFiltersTheListWasShowing(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-filters@example.com")

	notes := noteColumn(t, exportRecords(t, getExport(t, router, f.cookie, "?type=income")))

	if len(notes) != 1 || notes[0] != "August payslip" {
		t.Errorf("export of ?type=income = %q, want just the one income row", notes)
	}
}

func TestExportStartsWithAByteOrderMarkSoExcelReadsItAsUTF8(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-bom@example.com")

	body := getExport(t, router, f.cookie, "").Body.String()

	if !strings.HasPrefix(body, "\ufeff") {
		t.Errorf("export body starts %q, want a UTF-8 BOM", firstRunes(body, 8))
	}
}

func TestExportOffersTheFileAsADownloadNamedForTheMonth(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-filename@example.com")

	rec := getExport(t, router, f.cookie, "?month=2026-03")

	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="spend-2026-03.csv"` {
		t.Errorf("Content-Disposition = %q, want the month's own filename", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/csv; charset=utf-8", got)
	}
}

func firstRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		runes = runes[:n]
	}
	return string(runes)
}

// A note is free text, so it can contain the delimiter and the quote
// character. encoding/csv escapes both; a hand-rolled strings.Join would
// silently split one transaction across two columns, which is why this
// asserts on the parsed round-trip rather than on the raw body.
func TestExportEscapesANoteContainingCommasAndQuotes(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-escaping@example.com")

	const awkward = `Lunch, drinks, and a "tip"`
	if _, err := deps.Queries.CreateTransaction(context.Background(), sqlcgen.CreateTransactionParams{
		UserID: f.userID, CategoryID: f.foodID, Amount: 75000,
		Type: "expense", Description: awkward, OccurredOn: f.firstOfMonth,
	}); err != nil {
		t.Fatalf("seed awkward note: %v", err)
	}

	notes := noteColumn(t, exportRecords(t, getExport(t, router, f.cookie, "")))

	var found bool
	for _, note := range notes {
		if note == awkward {
			found = true
		}
	}
	if !found {
		t.Errorf("exported notes = %q, want one reading exactly %q", notes, awkward)
	}
}

// The link has to opt out of hx-boost. <body> carries hx-boost="true", so
// without this htmx intercepts the click, fetches the CSV over XHR and tries
// to swap it into the page -- the file never reaches the download manager
// and the user sees raw CSV where their transactions were.
func TestTheExportLinkOptsOutOfHxBoostSoTheBrowserDownloadsIt(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-link@example.com")

	body := getTransactions(t, router, f.cookie, "?type=income")

	link := regexp.MustCompile(`<a[^>]*href="/transactions/export\?[^"]*"[^>]*>`).FindString(body)
	if link == "" {
		t.Fatal("no link to /transactions/export on the transactions page")
	}
	if !strings.Contains(link, `hx-boost="false"`) {
		t.Errorf("export link is %s, want hx-boost=\"false\" on it", link)
	}
	if !strings.Contains(link, "type=income") {
		t.Errorf("export link is %s, want it to carry the active filters", link)
	}
}

// Below md the sticky page header already carries a title, a month picker
// and an add button, so the export goes under the list instead -- a fourth
// control in that row is what overflows a 375px screen. Both copies render
// from one "export_link" block, so this checks the mobile one is really
// there and really hidden from desktop, not that the markup was pasted
// twice and drifted.
func TestTheExportLinkIsReachableOnMobileWithoutCrowdingTheStickyHeader(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	f := seedFilterFixture(t, deps, router, "export-mobile@example.com")

	body := getTransactions(t, router, f.cookie, "")

	wrapper := regexp.MustCompile(`(?s)<div data-export-mobile class="([^"]*)">(.*?)</div>`).FindStringSubmatch(body)
	if wrapper == nil {
		t.Fatal("no element carrying data-export-mobile: the export is desktop-only")
	}
	if !strings.Contains(wrapper[1], "md:hidden") {
		t.Errorf("mobile export wrapper classes are %q, want md:hidden so it does not double up on desktop", wrapper[1])
	}
	if !strings.Contains(wrapper[2], `href="/transactions/export?`) || !strings.Contains(wrapper[2], `hx-boost="false"`) {
		t.Errorf("mobile export wrapper holds %q, want the boost-opted-out export link", wrapper[2])
	}
}

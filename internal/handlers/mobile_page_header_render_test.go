package handlers_test

import (
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// renderMobilePageHeader executes the shared sticky mobile header partial on
// its own: the data is a plain map because the production Data structs are
// unexported, and what's under test is the markup a given ActiveNav/month
// state produces.
func renderMobilePageHeader(t *testing.T, data map[string]any) string {
	t.Helper()
	tmpl := template.Must(template.New("mobile_header.html").
		Funcs(handlers.TemplateFuncs()).
		ParseFiles("../web/templates/mobile_header.html"))
	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "mobile_page_header", data); err != nil {
		t.Fatalf("execute mobile_page_header: %v", err)
	}
	return sb.String()
}

func headerData(activeNav string) map[string]any {
	return map[string]any{
		"ActiveNav":         activeNav,
		"MonthLabel":        "August 2026",
		"CurrentMonthValue": "2026-08",
		"AvailableMonths": []map[string]any{
			{"Value": "2026-07", "Label": "July 2026"},
		},
	}
}

// The transactions header is the one that gained the relocated add button:
// it must still reach the exact sheet the old page-bottom button opened, not
// a new endpoint, since this is a presentation-only move.
func TestMobilePageHeaderTransactionsHasAddButtonAndMonthPicker(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("transactions"))
	if !strings.Contains(out, "Transactions") {
		t.Error("header does not show the Transactions title")
	}
	if !strings.Contains(out, `aria-label="Add transaction"`) {
		t.Error("add button is missing its aria-label")
	}
	if !strings.Contains(out, `onclick="document.getElementById('mobile-add-sheet').showModal()"`) {
		t.Error("add button does not open the existing mobile-add-sheet the same way the old button did")
	}
	if !strings.Contains(out, "August 2026") {
		t.Error("month picker trigger does not show the current month label")
	}
	if !strings.Contains(out, `hx-target="#transactions-month-section"`) {
		t.Error("month picker does not target the transactions month section, so a month switch would not refresh it")
	}
}

// Categories' add button reuses openAddCategorySheet(), the existing DOM-move
// dialog opener -- not a new hx-get, since none exists today.
func TestMobilePageHeaderCategoriesHasAddButtonNoMonthPicker(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("categories"))
	if !strings.Contains(out, "Categories") {
		t.Error("header does not show the Categories title")
	}
	if !strings.Contains(out, `aria-label="Add category"`) {
		t.Error("add button is missing its aria-label")
	}
	if !strings.Contains(out, `onclick="openAddCategorySheet()"`) {
		t.Error("add button does not reuse the existing openAddCategorySheet trigger")
	}
	if strings.Contains(out, "▾") {
		t.Error("categories header should not render a month picker")
	}
}

func TestMobilePageHeaderDashboardHasMonthPickerNoAddButton(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("dashboard"))
	if !strings.Contains(out, "August 2026") {
		t.Error("header does not show the month as its title")
	}
	if !strings.Contains(out, `hx-target="#dashboard-month-section"`) {
		t.Error("month picker does not target the dashboard month section, so a month switch would not refresh it")
	}
	if strings.Contains(out, `aria-label="Add`) {
		t.Error("dashboard header should not render an add button")
	}
}

func TestMobilePageHeaderAddButtonMeetsMinimumTouchTarget(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("transactions"))
	if !strings.Contains(out, "w-11 h-11") {
		t.Error("add button's tappable area is smaller than the 44x44 minimum")
	}
	if !strings.Contains(out, "w-[34px] h-[34px]") {
		t.Error("add button's visible circle is not 34x34")
	}
}

// Only the empty state's own trigger should remain in transactions.html; the
// page-bottom button relocated into the sticky mobile header.
func TestTransactionsPageBottomAddButtonRemoved(t *testing.T) {
	body, err := os.ReadFile("../web/templates/transactions.html")
	if err != nil {
		t.Fatalf("read transactions.html: %v", err)
	}
	if got := strings.Count(string(body), "showModal()"); got != 1 {
		t.Errorf("transactions.html calls showModal() %d times, want 1 (the empty-state button only)", got)
	}
	if strings.Contains(string(body), "h-[46px]") {
		t.Error("the page-bottom Add transaction button (h-[46px]) is still present")
	}
}

// categories.js should keep the openAddCategorySheet() function (the shared
// header's + button, defined in mobile_header.html, calls it) but drop the
// page-bottom button that used to trigger it directly from this file.
func TestCategoriesPageBottomAddButtonRemoved(t *testing.T) {
	body, err := os.ReadFile("../web/templates/categories.html")
	if err != nil {
		t.Fatalf("read categories.html: %v", err)
	}
	if strings.Contains(string(body), "h-[46px]") {
		t.Error("the page-bottom Add category button (h-[46px]) is still present")
	}
	if strings.Contains(string(body), `onclick="openAddCategorySheet()"`) {
		t.Error("categories.html still triggers openAddCategorySheet() directly; that button should have moved into the shared mobile header")
	}
}

func TestMainContentClearsOnlyTheBottomNavOnMobile(t *testing.T) {
	body, err := os.ReadFile("../web/templates/layout.html")
	if err != nil {
		t.Fatalf("read layout.html: %v", err)
	}
	if !strings.Contains(string(body), "pb-[76px]") {
		t.Error("<main> no longer reserves pb-[76px] for the bottom nav now that the page-bottom button is gone")
	}
	if strings.Contains(string(body), "pb-24") {
		t.Error("<main> still carries the old pb-24, sized to leave room for the removed page-bottom button")
	}
}

var navMobileHeaderRe = regexp.MustCompile(`(?s)\{\{define "nav_mobile_header"\}\}(.*?)\{\{end\}\}`)

func TestMobileTopBarBlendsIntoTheAppBackground(t *testing.T) {
	body, err := os.ReadFile("../web/templates/mobile_header.html")
	if err != nil {
		t.Fatalf("read mobile_header.html: %v", err)
	}
	m := navMobileHeaderRe.FindStringSubmatch(string(body))
	if m == nil {
		t.Fatal("nav_mobile_header block not found in mobile_header.html")
	}
	if !strings.Contains(m[1], "bg-app") {
		t.Error("mobile top bar no longer uses the bg-app token, so it won't blend into the new sticky header below it")
	}
}

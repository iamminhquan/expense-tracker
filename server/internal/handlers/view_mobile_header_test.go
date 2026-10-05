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
		ParseFiles("../web/templates/mobile_header.html", "../web/templates/month_picker.html"))
	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "mobile_page_header", data); err != nil {
		t.Fatalf("execute mobile_page_header: %v", err)
	}
	return sb.String()
}

func headerData(activeNav string) map[string]any {
	return map[string]any{
		"ActiveNav":         activeNav,
		"Greeting":          "Good evening, immq",
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
	if !strings.Contains(out, `hx-target="#dashboard-month-section"`) {
		t.Error("month picker does not target the dashboard month section, so a month switch would not refresh it")
	}
	if strings.Contains(out, `aria-label="Add`) {
		t.Error("dashboard header should not render an add button")
	}
}

// The dashboard heading used to be the month, which the picker beside it
// already said. The greeting replaced it, so the month must survive in
// exactly one place: the picker trigger.
func TestMobilePageHeaderDashboardGreetsInsteadOfRepeatingTheMonth(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("dashboard"))
	if !strings.Contains(out, "Good evening, immq") {
		t.Error("dashboard header does not show the greeting as its title")
	}
	if got := strings.Count(out, "August 2026"); got != 1 {
		t.Errorf("dashboard header renders the month %d times, want 1 (the picker trigger only)", got)
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

// The mobile bar is sticky and in flow at the end of <body>, not fixed: that
// is what keeps its gap to the last card the same 14px on a long page and on
// one too short to scroll. Its own margins are the whole clearance, so <main>
// must not add a bottom reserve on mobile -- an earlier pb-[76px] there was
// 8px short of a fixed bar and tucked the last card behind it, and any reserve
// now would just re-open the gap on the dashboard.
func TestMobileBottomNavIsStickyAndCarriesItsOwnClearance(t *testing.T) {
	nav, err := os.ReadFile("../web/templates/nav.html")
	if err != nil {
		t.Fatalf("read nav.html: %v", err)
	}
	m := regexp.MustCompile(`(?s)\{\{define "nav_mobile"\}\}(.*?)\{\{end\}\}`).FindStringSubmatch(string(nav))
	if m == nil {
		t.Fatal("nav_mobile block not found in nav.html")
	}
	for _, want := range []string{"sticky", "bottom-4", "mt-[14px]", "mb-4", "mx-4"} {
		if !strings.Contains(m[1], want) {
			t.Errorf("nav_mobile lost %q; it needs all of sticky/bottom-4/mt-[14px]/mb-4/mx-4 to sit 14px under the last card on a short page and 16px off the bottom edge on a long one", want)
		}
	}
	if strings.Contains(m[1], "fixed") {
		t.Error("nav_mobile is fixed again, which strands the whole rest of the screen between the last card and the bar on a page too short to scroll")
	}

	layout, err := os.ReadFile("../web/templates/layout.html")
	if err != nil {
		t.Fatalf("read layout.html: %v", err)
	}
	for _, unwanted := range []string{"pb-[98px]", "pb-[76px]", "pb-24"} {
		if strings.Contains(string(layout), unwanted) {
			t.Errorf("<main> carries %s, a bottom reserve that stacks on top of the sticky nav's own margins", unwanted)
		}
	}
}

// A page shorter than the screen (the dashboard on a quiet month) used to
// leave a dead strip under the sticky bar. <body> is a full-height column and
// <main> grows, so the leftover height reaches the dashboard's chart card and
// the page ends where the screen does. Two things are easy to break here: the
// w-full on each page's content wrapper (mx-auto on a flex child otherwise
// shrinks the box to fit its content instead of stretching it, which quietly
// narrows every page), and the absolutely positioned bar canvas (an in-flow
// one is both sized from its parent by Chart.js and measured by that parent,
// so the card climbs off the screen on every resize).
func TestShortPagesFillTheScreenInsteadOfStrandingTheNav(t *testing.T) {
	layout, err := os.ReadFile("../web/templates/layout.html")
	if err != nil {
		t.Fatalf("read layout.html: %v", err)
	}
	for _, want := range []string{"min-h-screen flex flex-col", "grow flex flex-col"} {
		if !strings.Contains(string(layout), want) {
			t.Errorf("layout.html lost %q; without it <main> has no leftover height to hand down and a short page strands the nav mid-screen", want)
		}
	}

	for _, page := range []string{"dashboard.html", "transactions.html", "categories.html"} {
		body, err := os.ReadFile("../web/templates/" + page)
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		if !strings.Contains(string(body), `w-full max-w-[880px] mx-auto`) {
			t.Errorf("%s's content wrapper is missing w-full; as a flex child of <main>, mx-auto shrinks it to its content width and narrows the whole page", page)
		}
	}

	dashboard, err := os.ReadFile("../web/templates/dashboard.html")
	if err != nil {
		t.Fatalf("read dashboard.html: %v", err)
	}
	if !strings.Contains(string(dashboard), `<canvas id="bar-chart" class="absolute inset-0`) {
		t.Error("the bar chart canvas is back in flow; Chart.js sizes it from the parent that measures it, so each resize feeds the next")
	}
	if !strings.Contains(string(dashboard), `relative grow min-h-[118px] md:min-h-[158px]`) {
		t.Error("the bar chart's wrapper lost its grow/min-h, so the chart no longer absorbs the leftover screen height")
	}
	if !strings.Contains(string(dashboard), "grid-rows-[auto_1fr] md:grid-rows-none") {
		t.Error("the chart pair lost grid-rows-[auto_1fr]: without it the leftover height is split between both cards instead of going to the bar chart")
	}

	charts, err := os.ReadFile("../web/static/charts.js")
	if err != nil {
		t.Fatalf("read charts.js: %v", err)
	}
	if !strings.Contains(string(charts), "maintainAspectRatio: false") {
		t.Error("charts.js dropped maintainAspectRatio: false, so the bar chart computes a height from its width and overflows the card it is meant to fill")
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

// The settings header is the plainest of the four: a title and nothing else.
// There is no month to browse and no row to add, so an empty <h1> -- what
// an unrecognised ActiveNav produces -- would leave the page nameless on
// mobile, where the desktop heading is hidden.
func TestMobilePageHeaderSettingsShowsTitleOnly(t *testing.T) {
	out := renderMobilePageHeader(t, headerData("settings"))
	if !strings.Contains(out, "Settings") {
		t.Error("header does not show the Settings title")
	}
	if strings.Contains(out, "August 2026") {
		t.Error("settings header should not carry the month picker")
	}
	if strings.Contains(out, "aria-label=\"Add") {
		t.Error("settings header should not carry an add button")
	}
}

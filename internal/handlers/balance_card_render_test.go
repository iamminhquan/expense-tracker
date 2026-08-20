package handlers_test

import (
	"html"
	"html/template"
	"regexp"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// renderBalanceCard executes the shared partial on its own. The data is a
// map rather than the handlers.balanceCard the production code passes,
// because that type is unexported -- what is under test here is the markup
// the template produces for a given set of already-computed values.
func renderBalanceCard(t *testing.T, data map[string]any) string {
	t.Helper()
	tmpl := template.Must(template.New("balance_card.html").
		Funcs(handlers.TemplateFuncs()).
		ParseFiles("../web/templates/balance_card.html"))
	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "balance_card", data); err != nil {
		t.Fatalf("execute balance_card: %v", err)
	}
	return sb.String()
}

// renderBalanceText renders the card and decodes the HTML entities
// html/template emits, so an assertion can be written the way the number
// reads on screen. Without this a "+" comes back as "&#43;".
func renderBalanceText(t *testing.T, data map[string]any) string {
	t.Helper()
	return html.UnescapeString(renderBalanceCard(t, data))
}

func balanceData(overrides map[string]any) map[string]any {
	data := map[string]any{
		"Expense": int64(5800000), "Income": int64(10000000), "Remaining": int64(4200000),
		"Variant": "transactions", "SpentPct": 58, "LeftPct": 42, "ShowLeftPct": true,
		"Label": "Left in August", "RatioLabel": "Spent 58% of this month's income",
		"Empty": false, "OOB": false,
	}
	for k, v := range overrides {
		data[k] = v
	}
	return data
}

var fillWidthRe = regexp.MustCompile(`width:\s*(\d+)%`)

// The bar is the one number that reaches the DOM as a style rather than as
// text. html/template filters values it cannot parse in a CSS context down
// to ZgotmplZ, so this also proves the percentage survives the trip.
func TestBalanceCardBarFillMatchesSpentPercent(t *testing.T) {
	out := renderBalanceCard(t, balanceData(nil))
	m := fillWidthRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no bar fill width found in rendered card:\n%s", out)
	}
	if m[1] != "58" {
		t.Errorf("bar fill width = %s%%, want 58%%", m[1])
	}
	if strings.Contains(out, "ZgotmplZ") {
		t.Error("the width was escaped to ZgotmplZ; the percentage never reached the style attribute")
	}
}

func TestBalanceCardOverspendingFillsTheBarAndDropsTheLeftPercent(t *testing.T) {
	out := renderBalanceCard(t, balanceData(map[string]any{
		"Expense": int64(12000000), "Remaining": int64(-2000000), "SpentPct": 100,
		"LeftPct": 0, "ShowLeftPct": false, "RatioLabel": "Over this month's income",
	}))
	if m := fillWidthRe.FindStringSubmatch(out); m == nil || m[1] != "100" {
		t.Errorf("overspent bar fill = %v, want 100%%", m)
	}
	if strings.Contains(out, "% left") {
		t.Error(`card still shows a "% left" figure after spending passed income`)
	}
	if !strings.Contains(html.UnescapeString(out), "Over this month's income") {
		t.Error("card does not show the over-income caption")
	}
}

// A negative balance is the card's alarm state and has to read differently
// from a healthy one -- same token the transaction rows use for expenses.
func TestBalanceCardColoursNegativeRemainingAsExpense(t *testing.T) {
	negative := renderBalanceText(t, balanceData(map[string]any{
		"Remaining": int64(-1200000), "SpentPct": 100, "ShowLeftPct": false,
	}))
	if !strings.Contains(negative, "-1,200,000₫") {
		t.Error("negative balance is not rendered with its sign")
	}
	if !strings.Contains(negative, "text-expense") {
		t.Error("negative balance is not coloured with the expense token")
	}

	positive := renderBalanceText(t, balanceData(nil))
	if !strings.Contains(positive, "+4,200,000₫") {
		t.Error("positive balance is not rendered with its sign")
	}
}

// An untouched month still renders the card; it just greys the zero out
// rather than presenting it as a real balance.
func TestBalanceCardEmptyMonthStillRendersAGreyedZero(t *testing.T) {
	out := renderBalanceCard(t, balanceData(map[string]any{
		"Expense": int64(0), "Income": int64(0), "Remaining": int64(0),
		"SpentPct": 0, "LeftPct": 0, "ShowLeftPct": false,
		"RatioLabel": "No income this month", "Empty": true,
	}))
	if !strings.Contains(out, "text-ink-zero") {
		t.Error("empty month's zero is not greyed with the ink-zero token")
	}
	if !strings.Contains(out, "0₫") {
		t.Error("empty month does not show 0₫")
	}
	if m := fillWidthRe.FindStringSubmatch(out); m == nil || m[1] != "0" {
		t.Errorf("empty month bar fill = %v, want 0%%", m)
	}
}

func TestBalanceCardEmitsOOBAttributeOnlyWhenFlagged(t *testing.T) {
	if out := renderBalanceCard(t, balanceData(nil)); strings.Contains(out, "hx-swap-oob") {
		t.Error("in-page render carries hx-swap-oob; the month fragment would fight itself")
	}
	out := renderBalanceCard(t, balanceData(map[string]any{"OOB": true}))
	if !strings.Contains(out, `hx-swap-oob="true"`) {
		t.Error("OOB render is missing hx-swap-oob, so mutations would not update the card")
	}
	if strings.Count(out, `id="balance-card"`) != 1 {
		t.Errorf(`rendered card has %d id="balance-card" roots, want 1`, strings.Count(out, `id="balance-card"`))
	}
}

// The dashboard's card sits in a row beside its own Spent/Earned cards, so
// repeating those two figures inside it would print them twice side by side.
func TestBalanceCardDashboardVariantOmitsTheDetailRow(t *testing.T) {
	transactions := renderBalanceCard(t, balanceData(nil))
	if !strings.Contains(transactions, "5,800,000₫") || !strings.Contains(transactions, "10,000,000₫") {
		t.Error("transactions card is missing its Spent/Earned detail figures")
	}

	dashboard := renderBalanceCard(t, balanceData(map[string]any{"Variant": "dashboard", "Label": "Left this month"}))
	if strings.Contains(dashboard, "5,800,000₫") {
		t.Error("dashboard card repeats the Spent figure its sibling card already shows")
	}
	if !strings.Contains(dashboard, "Left this month") {
		t.Error("dashboard card is missing its label")
	}
}

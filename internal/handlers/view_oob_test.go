package handlers_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The header widget is rendered from three places -- both nav bars and the
// out-of-band fragment every mutation returns -- so all three go through one
// {{define}}. Duplicating the markup instead would let the copies drift, and
// a widget that renders differently after a mutation than it did on page load
// is exactly the bug this shape prevents.
func TestHeaderBalanceHasExactlyOneDefinition(t *testing.T) {
	dir := "../web/templates"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}

	defineRe := regexp.MustCompile(`{{\s*define\s+"header_balance"\s*}}`)
	var found []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for range defineRe.FindAllString(string(body), -1) {
			found = append(found, e.Name())
		}
	}

	if len(found) != 1 {
		t.Fatalf(`{{define "header_balance"}} appears %d times (%v), want exactly 1`, len(found), found)
	}
	if found[0] != "header_balance.html" {
		t.Errorf(`"header_balance" defined in %s, want header_balance.html`, found[0])
	}
}

// Both nav bars render the widget and both sit in the DOM at once, so a swap
// that names only one id leaves the other stale at whichever breakpoint the
// user is on. Each id has to appear twice across the shared partials: once as
// the in-page target in the bar that renders it (nav.html for desktop,
// mobile_header.html for mobile) and once in header_balance.html's
// out-of-band fragment, or the swap has nothing to land on.
func TestHeaderBalanceOOBReachesBothNavBars(t *testing.T) {
	shared := []string{"nav.html", "mobile_header.html", "header_balance.html"}
	var body string
	for _, name := range shared {
		body += readTemplate(t, name)
	}
	for _, id := range []string{"header-balance-desktop", "header-balance-mobile"} {
		if got := strings.Count(body, `id="`+id+`"`); got != 2 {
			t.Errorf(`id=%q appears %d times across %v, want 2 (the in-page target and the OOB swap)`, id, got, shared)
		}
	}
}

// The widget itself must never carry hx-swap-oob: it is rendered in-page by
// both nav bars, and an unconditional attribute there would have htmx lift it
// out of the page it is part of. Only the wrapper the mutation fragment sends
// is marked for swapping.
func TestHeaderBalanceIsMarkedOOBOnlyByItsWrapper(t *testing.T) {
	body := readTemplate(t, "header_balance.html")
	widget := body[strings.Index(body, `{{define "header_balance"}}`):]
	widget = widget[:strings.Index(widget, "{{end}}")]
	if strings.Contains(widget, "hx-swap-oob") {
		t.Error(`the header_balance widget carries hx-swap-oob itself; only header_balance_oob may`)
	}
	if !strings.Contains(body, `{{define "header_balance_oob"}}`) {
		t.Error("no header_balance_oob wrapper; mutations could not refresh the widget")
	}
}

// The balance used to be spelled out in three other places on the
// transactions page. The nav widget replaces all of them -- leaving one
// behind means two numbers on screen that disagree the moment one stops
// updating.
func TestRetiredTotalsElementsAreGone(t *testing.T) {
	retired := []string{"remaining-row", "mobile-stat-cards", "totals-summary", "balance-card"}
	dir := "../web/templates"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		body := readTemplate(t, e.Name())
		for _, id := range retired {
			if strings.Contains(body, `id="`+id+`"`) {
				t.Errorf(`%s still renders id=%q, which the nav balance widget replaced`, e.Name(), id)
			}
		}
	}
}

func readTemplate(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("../web/templates", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(body)
}

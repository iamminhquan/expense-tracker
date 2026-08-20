package handlers_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The balance card is rendered in-page and again as an OOB fragment after
// every mutation. Both go through one {{define}}, so the two copies cannot
// drift apart the way #totals-summary's hand-duplicated markup used to --
// this test just holds that single-definition property in place.
func TestBalanceCardHasExactlyOneDefinition(t *testing.T) {
	dir := "../web/templates"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}

	defineRe := regexp.MustCompile(`{{\s*define\s+"balance_card"\s*}}`)
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
		t.Fatalf(`{{define "balance_card"}} appears %d times (%v), want exactly 1`, len(found), found)
	}
	if found[0] != "balance_card.html" {
		t.Errorf(`"balance_card" defined in %s, want balance_card.html`, found[0])
	}
}

// Every mutation handler swaps the card by id, and the month picker re-renders
// it inside the month section, so the id has to be fixed rather than derived
// from anything in the data.
func TestBalanceCardRootCarriesTheFixedID(t *testing.T) {
	body := readTemplate(t, "balance_card.html")
	if !strings.Contains(body, `id="balance-card"`) {
		t.Error(`balance_card.html has no id="balance-card" root`)
	}
}

// hx-swap-oob must be conditional. The month picker's response is itself an
// htmx swap containing this card; if the attribute were unconditional, htmx
// would pull the card out of that fragment and swap it into the element the
// same fragment is in the middle of replacing.
func TestBalanceCardMarksItselfOOBOnlyWhenAsked(t *testing.T) {
	body := readTemplate(t, "balance_card.html")
	// Matches the attribute itself, not the word where the comment above it
	// explains why the attribute is conditional.
	oobRe := regexp.MustCompile(`hx-swap-oob="`)
	for _, m := range oobRe.FindAllStringIndex(body, -1) {
		window := body[max(0, m[0]-40):m[0]]
		if !strings.Contains(window, "{{if .OOB}}") {
			t.Errorf("hx-swap-oob at offset %d is not guarded by {{if .OOB}}: %q", m[0], window)
		}
	}
	if !strings.Contains(body, "{{if .OOB}}") {
		t.Error("balance_card.html never emits hx-swap-oob; mutations could not update it")
	}
}

// The balance used to be spelled out in three other places on the
// transactions page. The card replaces all of them -- leaving one behind
// means two numbers on screen that disagree the moment one stops updating.
func TestRetiredTotalsElementsAreGone(t *testing.T) {
	retired := []string{"remaining-row", "mobile-stat-cards", "totals-summary"}
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
				t.Errorf(`%s still renders id=%q, which the balance card replaced`, e.Name(), id)
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

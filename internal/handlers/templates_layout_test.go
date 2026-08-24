package handlers_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	formControlRe = regexp.MustCompile(`(?s)<(input|select|textarea)\b[^>]*>`)
	classRe       = regexp.MustCompile(`class="([^"]*)"`)
)

// A flex item's automatic minimum size is the width its own content needs, so
// a form control marked only `flex-1` refuses to shrink past that and pushes
// its row off the screen -- which is how the mobile add sheet's date and note
// fields ended up hanging over the right edge of a 375px phone. Pair flex-1
// with a min-width (min-w-0 to shrink freely, min-w-[Npx] to wrap instead).
func TestFlexibleFormControlsDeclareAMinWidth(t *testing.T) {
	dir := "../web/templates"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}

	var offenders []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, tag := range formControlRe.FindAllString(string(body), -1) {
			m := classRe.FindStringSubmatch(tag)
			if m == nil {
				continue
			}
			classes := strings.Fields(m[1])
			var flexible, bounded bool
			for _, c := range classes {
				// a responsive prefix still applies at that breakpoint
				if _, bare, found := strings.Cut(c, ":"); found {
					c = bare
				}
				if c == "flex-1" || c == "flex-auto" {
					flexible = true
				}
				if strings.HasPrefix(c, "min-w-") || strings.HasPrefix(c, "w-") {
					bounded = true
				}
			}
			if flexible && !bounded {
				offenders = append(offenders, e.Name()+": "+tag)
			}
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("these form controls can grow but never shrink, so they overflow their row:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// The bottom sheet's grab handle is inert markup on its own -- the drag lives
// in app.js and finds the handle by attribute. Renaming one side without the
// other leaves a pill that looks draggable and is not, which is the bug this
// behaviour was added to fix. The handle also has to opt out of the browser's
// own touch gestures, or a downward drag scrolls the page instead.
func TestBottomSheetHandleStaysWiredToTheDragScript(t *testing.T) {
	sheet, err := os.ReadFile("../web/templates/transactions.html")
	if err != nil {
		t.Fatalf("read transactions.html: %v", err)
	}
	script, err := os.ReadFile("../web/static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}

	handle := regexp.MustCompile(`<div data-sheet-handle class="([^"]*)"`).FindSubmatch(sheet)
	if handle == nil {
		t.Fatal("no element carrying data-sheet-handle in transactions.html: the add sheet has no draggable grab handle")
	}
	if !strings.Contains(string(handle[1]), "touch-none") {
		t.Fatalf("the grab handle needs touch-none, or the browser scrolls instead of dragging; got %q", handle[1])
	}
	if !strings.Contains(string(script), "[data-sheet-handle]") {
		t.Fatal("app.js no longer looks for [data-sheet-handle]: the handle is decorative again")
	}
}

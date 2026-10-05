package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
	"expensetracker/internal/web"
)

// Every page set must carry these on top of its own blocks: the shell, the
// nav bars, and the balance widget both nav bars render. A page whose file
// list lost one of them still parses -- html/template only resolves a
// {{template}} call when it executes -- so the failure would otherwise
// surface as a 500 on that one page at runtime.
var sharedBlocks = []string{
	"layout",
	"content",
	"logo_nav",
	"user_menu",
	"header_balance",
	"header_balance_oob",
	"nav_desktop",
	"nav_mobile",
	"nav_mobile_header",
	"mobile_page_header",
}

func TestTemplatesDefinesSharedBlocksInEveryPageSet(t *testing.T) {
	sets, err := web.Templates(handlers.TemplateFuncs())
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	for _, page := range []string{"auth", "categories", "transactions", "dashboard"} {
		tmpl, ok := sets[page]
		if !ok {
			t.Errorf("no template set registered for page %q", page)
			continue
		}
		for _, block := range sharedBlocks {
			if tmpl.Lookup(block) == nil {
				t.Errorf("page %q is missing the %q block", page, block)
			}
		}
	}
}

// The fragment names handlers reach for through renderNamed. A rename in a
// template without the matching rename in the handler is a 500 on one htmx
// interaction, which no other test in this package would notice.
func TestTemplatesDefinesEveryFragmentHandlersRender(t *testing.T) {
	sets, err := web.Templates(handlers.TemplateFuncs())
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	fragments := map[string][]string{
		"auth": {"auth_card_body"},
		"categories": {
			"add_category_form", "category_row", "category_row_edit",
			"category_create_response", "categories_empty_oob",
		},
		"transactions": {
			"transactions_month_section", "quick_add_form", "mobile_quick_add_form",
			"category_options", "category_chips", "transaction_row",
			"transaction_row_edit", "transaction_row_delete_confirm",
			"transaction_create_response", "totals_oob",
		},
		"dashboard": {"dashboard_month_section"},
	}

	for page, names := range fragments {
		for _, name := range names {
			if sets[page].Lookup(name) == nil {
				t.Errorf("page %q is missing the %q fragment", page, name)
			}
		}
	}
}

// Every asset the templates reference by URL. A file renamed on disk without
// the matching edit in the template would otherwise ship a page whose
// stylesheet or script 404s -- visible only by loading it in a browser.
func TestStaticHandlerServesEveryReferencedAsset(t *testing.T) {
	want := map[string]string{
		"app.css":            "text/css",
		"app.js":             "javascript",
		"tailwind-config.js": "javascript",
		"charts.js":          "javascript",
		"categories.js":      "javascript",
	}

	handler := web.StaticHandler()
	for name, contentType := range want {
		req := httptest.NewRequest(http.MethodGet, web.StaticPrefix+name, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s%s = %d, want 200", web.StaticPrefix, name, rec.Code)
			continue
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, contentType) {
			t.Errorf("GET %s%s Content-Type = %q, want it to contain %q", web.StaticPrefix, name, got, contentType)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("GET %s%s served an empty body", web.StaticPrefix, name)
		}
	}
}

// The ETag is the only thing that can make a repeat request cheap: embed.FS
// reports a zero ModTime, so there is no Last-Modified to revalidate against.
func TestStaticHandlerAnswers304ForAMatchingETag(t *testing.T) {
	handler := web.StaticHandler()

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, web.StaticPrefix+"app.css", nil))
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the first response, so every repeat visit refetches the whole file")
	}

	req := httptest.NewRequest(http.MethodGet, web.StaticPrefix+"app.css", nil)
	req.Header.Set("If-None-Match", etag)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Errorf("repeat request with If-None-Match = %d, want 304", second.Code)
	}
}

func TestStaticHandler404sOnAnUnknownAsset(t *testing.T) {
	rec := httptest.NewRecorder()
	web.StaticHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, web.StaticPrefix+"nope.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET an asset that does not exist = %d, want 404", rec.Code)
	}
}

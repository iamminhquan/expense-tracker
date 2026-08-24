package web_test

import (
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

// Package web owns everything the browser is served: the html/template
// files under templates/ and, alongside them, the CSS and JavaScript under
// static/.
//
// Both trees are embedded into the binary, so the server does not depend on
// being started from the repo root to find them. (The migrations under
// internal/database are still read from disk at startup -- see
// cmd/server/main.go -- so the working directory still matters for those.)
package web

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates
var templatesFS embed.FS

// sharedTemplates are parsed into every page's set: the page shell, the nav
// bars, and the balance widget the nav bars render. A page that forgot one
// of these would fail at execute time with "no such template", not at
// startup, so they are listed once here rather than repeated per page.
var sharedTemplates = []string{"layout.html"}

// pageTemplates maps a page name -- the key handlers use when they call
// render/renderNamed -- to the files that page's set is built from, on top
// of sharedTemplates.
//
// Each page gets its own *template.Template rather than one set holding
// everything, because several pages define blocks under the same name
// ("content" above all) and a single set would let the last one parsed win.
var pageTemplates = map[string][]string{
	"auth":         {"auth.html", "auth_card_body.html"},
	"categories":   {"categories.html", "category_row.html"},
	"transactions": {"transactions.html", "transaction_row.html"},
	"dashboard":    {"dashboard.html"},
}

// Templates parses every page set from the embedded templates directory.
//
// funcs is taken as an argument rather than imported, so this package stays
// free of any dependency on internal/handlers -- which is what lets the
// handlers package (and its tests) build the very same sets the server runs
// with, instead of each keeping its own copy of the file list.
func Templates(funcs template.FuncMap) (map[string]*template.Template, error) {
	sets := make(map[string]*template.Template, len(pageTemplates))
	for page, files := range pageTemplates {
		paths := make([]string, 0, len(sharedTemplates)+len(files))
		for _, name := range append(append([]string{}, sharedTemplates...), files...) {
			paths = append(paths, "templates/"+name)
		}
		tmpl, err := template.New("layout.html").Funcs(funcs).ParseFS(templatesFS, paths...)
		if err != nil {
			return nil, fmt.Errorf("parse %s templates: %w", page, err)
		}
		sets[page] = tmpl
	}
	return sets, nil
}

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
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// sharedTemplates are parsed into every page's set: the page shell, the two
// nav bars, the mobile header, and the widgets and controls they render. A
// page that forgot one of these would fail at execute time with "no such
// template", not at startup, so they are listed once here rather than
// repeated per page.
var sharedTemplates = []string{
	"layout.html",
	"nav.html",
	"mobile_header.html",
	"month_picker.html",
	"user_menu.html",
	"header_balance.html",
}

// pageTemplates maps a page name -- the key handlers use when they call
// render/renderNamed -- to the files that page's set is built from, on top
// of sharedTemplates.
//
// Each page gets its own *template.Template rather than one set holding
// everything, because several pages define blocks under the same name
// ("content" above all) and a single set would let the last one parsed win.
var pageTemplates = map[string][]string{
	"auth":            {"auth.html", "auth_card_body.html"},
	"forgot_password": {"forgot_password.html"},
	"reset_password":  {"reset_password.html"},
	"verify_email":    {"verify_email.html"},
	"categories":      {"categories.html", "category_row.html"},
	"transactions":    {"transactions.html", "transaction_form.html", "transaction_row.html", "transaction_filters.html"},
	"dashboard":       {"dashboard.html"},
	"import":          {"import.html"},
	"settings":        {"settings.html", "settings_inbox.html"},
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

// StaticPrefix is the URL path the static assets are served under. The
// templates hardcode it in their <link>/<script> tags, so it is exported to
// keep the router's route and those tags from drifting apart silently.
const StaticPrefix = "/static/"

// staticETag is one digest over the whole embedded tree, computed at init.
// Every asset shares it, which is fine: a browser stores an ETag per URL, so
// the only cost is that a deploy makes every asset revalidate once.
//
// It exists at all because embed.FS reports a zero ModTime: there is no
// Last-Modified to revalidate against, so an ETag is the only thing that can
// turn a repeat visit into a 304.
var staticETag = hashStaticTree()

func hashStaticTree() string {
	h := sha256.New()
	err := fs.WalkDir(staticFS, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := staticFS.ReadFile(p)
		if err != nil {
			return err
		}
		// Length-prefixed with the path, so moving bytes between two files
		// cannot leave the digest unchanged.
		fmt.Fprintf(h, "%s:%d:", p, len(body))
		h.Write(body)
		return nil
	})
	if err != nil {
		// Only reachable if the embedded tree itself is unreadable, which
		// would be a build-time problem, not a runtime one.
		panic("web: read embedded static assets: " + err.Error())
	}
	return `"` + hex.EncodeToString(h.Sum(nil)[:8]) + `"`
}

// StaticHandler serves the embedded static/ tree. It is mounted publicly:
// the CSS and JS are the same for every visitor, and the login page needs
// them before anyone is authenticated.
//
// Cache-Control is "no-cache" rather than a max-age, so a deploy never
// leaves a browser holding a stale app.js against fresh markup; the ETag is
// what keeps the repeat request cheap (a 304 with no body).
func StaticHandler() http.Handler {
	// The embedded tree is rooted at "static", which is exactly what
	// StaticPrefix names, so the request path needs no rewriting. FileServerFS
	// answers If-None-Match against the ETag set below and picks the
	// Content-Type off the extension.
	files := http.FileServerFS(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No directory listings: the only things served here are the files
		// the templates name.
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", staticETag)
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}

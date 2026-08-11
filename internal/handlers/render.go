package handlers

import (
	"bytes"
	"log"
	"net/http"
)

// render looks up the named page's template set in deps.Templates and
// executes it into a buffer first, only copying the result to w once that
// succeeds. ExecuteTemplate writes incrementally, so calling it directly
// against w risks a mid-render failure producing a 200 OK with a silently
// truncated body and no server-side trace. Buffering first means any error
// is caught before a single byte reaches the client, so we can log it and
// return a proper 500 instead.
func render(w http.ResponseWriter, deps Deps, page string, data any) {
	tmpl, ok := deps.Templates[page]
	if !ok {
		log.Printf("render: no template registered for page %q", page)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render: execute template %q: %v", page, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("render: write response for %q: %v", page, err)
	}
}

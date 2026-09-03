package handlers

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// currentURLQuery is the query string of the page that issued this request.
// A mutation POST/PATCH/DELETE carries none of its own, but htmx sends the
// originating page's full URL in HX-Current-URL.
//
// A missing or malformed header yields nil, and a read off nil url.Values
// answers "" -- the same lenient fallback the month, filter and page
// readers each used to spell out for themselves.
func currentURLQuery(r *http.Request) url.Values {
	u, err := url.Parse(r.Header.Get("HX-Current-URL"))
	if err != nil {
		return nil
	}
	return u.Query()
}

// idParam reads the {id} route parameter as an int64. Every row-scoped
// handler starts with it, and every one of them answers a malformed id the
// same way, so the 400 is written here and the caller only has to stop:
//
//	id, ok := idParam(w, r)
//	if !ok {
//		return
//	}
func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

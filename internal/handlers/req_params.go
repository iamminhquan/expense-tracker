package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

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

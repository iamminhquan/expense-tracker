package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgInt64 converts a plain int64 (e.g. an authenticated user's ID) into the
// pgtype.Int8 that sqlc generates for a nullable *_id column.
func pgInt64(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

// pgText wraps a string as the nullable text sqlc generates for a nullable
// column. An empty string is still a valid, non-NULL value here.
func pgText(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: true}
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

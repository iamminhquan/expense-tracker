package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgInt64 converts a plain int64 (e.g. an authenticated user's ID) into the
// pgtype.Int8 that sqlc generates for a nullable *_id column.
func pgInt64(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

// chiURLParam reads a chi route parameter from the request.
func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

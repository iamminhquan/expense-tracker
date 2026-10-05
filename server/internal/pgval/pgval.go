// Package pgval wraps plain Go values in the pgtype ones sqlc generates for
// nullable columns.
//
// Every one of these says the same thing -- "this value is present" -- and
// says it in three lines of struct literal at each call site. They lived in
// internal/handlers until the email processor needed them too; they need
// neither a request nor a database, so they belong outside it.
package pgval

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Int64 converts a plain int64 (e.g. an authenticated user's ID) into the
// pgtype.Int8 that sqlc generates for a nullable *_id column.
func Int64(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

// Text wraps a string as the nullable text sqlc generates for a nullable
// column. An empty string is still a valid, non-NULL value here.
func Text(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: true}
}

// Date converts a parsed calendar date into the pgtype.Date that sqlc
// generates for a DATE column.
func Date(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

package handlers

import (
	"net/http"
	"time"

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

// vietnamLocation is loaded once at package init and reused by every
// "current month" calculation. This app is for Vietnamese users (UTC+7);
// anchoring month boundaries to the server's UTC clock instead would make
// "this month" resolve to the previous month for roughly the first 7 hours
// of every month in server time, and could make a transaction a user just
// added (dated with their own local calendar date) fall outside the range
// the list/dashboard pages just queried. If the timezone database isn't
// available in the runtime environment, fall back to a fixed UTC+7 offset
// (Vietnam has no DST) rather than failing startup over this.
var vietnamLocation = loadVietnamLocation()

func loadVietnamLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}

// currentMonthRange returns the [from, to) pgtype.Date bounds for "this
// month" in Vietnam's timezone, shared by every handler that needs a
// current-month window (transactions list, dashboard totals/breakdown).
func currentMonthRange() (from, to pgtype.Date) {
	now := time.Now().In(vietnamLocation)
	fromTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, vietnamLocation)
	toTime := fromTime.AddDate(0, 1, 0)
	return pgDate(fromTime), pgDate(toTime)
}

package handlers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Everything that answers "which month is this page showing?". The app is
// month-scoped throughout -- the transactions list, the dashboard totals and
// the balance widget all take a [from, to) window -- so these live together
// rather than beside whichever handler happened to need one first.

// pgDate converts a parsed calendar date into the pgtype.Date that sqlc
// generates for a DATE column.
func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func monthLabel(t time.Time) string      { return t.Format("January 2006") }
func monthLabelLower(t time.Time) string { return t.Format("January") }

// monthRangeFor returns the [from, to) bounds for the "YYYY-MM" value the
// month dropdown sends via ?month=, falling back to the current Vietnam-
// local month when param is empty or malformed.
func monthRangeFor(param string) (from, to pgtype.Date) {
	t, err := time.ParseInLocation("2006-01", param, vietnamLocation)
	if err != nil {
		return currentMonthRange()
	}
	fromTime := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, vietnamLocation)
	return pgDate(fromTime), pgDate(fromTime.AddDate(0, 1, 0))
}

// monthRangeFromRequest determines which month a mutation response's OOB
// totals fragment should reflect: the month the page the request originated
// from was actually displaying, not necessarily today's month. A create/
// update/delete request's own URL never carries "?month=" (only the page's
// GET request does), but htmx sends the full URL of that originating page,
// query string included, in the HX-Current-URL header on every request it
// issues -- so that header, not r.URL, is where the active month lives.
// Falls back to currentMonthRange() (mirroring monthRangeFor's own
// empty/malformed fallback) when the header is absent or unparseable, e.g.
// non-htmx requests.
func monthRangeFromRequest(r *http.Request) (from, to pgtype.Date) {
	raw := r.Header.Get("HX-Current-URL")
	if raw == "" {
		return currentMonthRange()
	}
	u, err := url.Parse(raw)
	if err != nil {
		return currentMonthRange()
	}
	return monthRangeFor(u.Query().Get("month"))
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

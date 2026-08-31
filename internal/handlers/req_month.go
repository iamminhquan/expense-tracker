package handlers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"expensetracker/internal/pgval"
)

// Everything that answers "which month is this page showing?". The app is
// month-scoped throughout -- the transactions list, the dashboard totals and
// the balance widget all take a [from, to) window -- so these live together
// rather than beside whichever handler happened to need one first.

// monthOption is one entry in the month picker's dropdown: the "YYYY-MM" a
// link carries, and the "August 2026" it reads as.
type monthOption struct {
	Value string
	Label string
}

// monthOptions turns the months an account has transactions in into the
// picker's entries. The current month is left out on purpose: both pages
// that render the picker pin it as their own "This month" entry above the
// list, so including it here would offer the same month twice.
func monthOptions(months []pgtype.Date, current pgtype.Date) []monthOption {
	var options []monthOption
	for _, m := range months {
		if m.Time.Year() == current.Time.Year() && m.Time.Month() == current.Time.Month() {
			continue
		}
		options = append(options, monthOption{
			Value: m.Time.Format("2006-01"),
			Label: monthLabel(m.Time),
		})
	}
	return options
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
	return pgval.Date(fromTime), pgval.Date(fromTime.AddDate(0, 1, 0))
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
	return monthRangeFor(monthParamFromRequest(r))
}

// monthParamFromRequest pulls the raw "YYYY-MM" out of the originating page's
// URL, for the callers that have to hand it on rather than resolve it -- a
// create that re-renders the whole list section needs the month value the
// pager's own links will be built from. Returns "" when there is nothing to
// read, which every consumer treats as the current month.
func monthParamFromRequest(r *http.Request) string {
	raw := r.Header.Get("HX-Current-URL")
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Query().Get("month")
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
	return pgval.Date(fromTime), pgval.Date(toTime)
}

// allMonths is what ?month= carries when the transactions page is showing a
// whole history rather than one calendar month. It is spelled out here
// because three places have to agree on it: the picker item that sends it,
// newTxnScope that reads it, and the Value every link on the page renders it
// back into.
const allMonths = "all"

// The bounds the all-time scope reports. They are deliberately absurd rather
// than derived from the user's earliest transaction: the point is a window
// wide enough that "occurred_on >= from AND occurred_on < to" stops selecting
// anything out, which lets the list, the count and the export keep running
// the one query they already run instead of growing a second, month-less
// variant of each.
var (
	allTimeFrom = pgval.Date(time.Date(1, 1, 1, 0, 0, 0, 0, vietnamLocation))
	allTimeTo   = pgval.Date(time.Date(9999, 12, 31, 0, 0, 0, 0, vietnamLocation))
)

// txnScope is which slice of time the transactions page is showing: one
// calendar month, or every month there has ever been.
//
// It exists because the page cannot simply format its lower bound back into
// a "YYYY-MM" the way it used to. Every link the page builds -- the pager,
// the export, the URL pushed via HX-Push-Url, the month the filter controls
// file under -- has to name the scope it belongs to, and an all-time window
// formatted as a month reads "0001-01". So the scope carries the spelling it
// arrived as, and the bounds hang off it rather than the other way round.
//
// The dashboard is deliberately not a consumer: its charts are built month by
// month and mean nothing over a whole history, so it keeps calling
// monthRangeFor, which has never heard of allMonths and treats it as
// malformed like any other unparseable value.
type txnScope struct {
	Value string // "2026-08" or "all" -- what every link's ?month= carries
	Label string // "August 2026" or "All months" -- what the picker shows
	All   bool

	from, to pgtype.Date
}

// newTxnScope reads the scope out of a raw ?month= value. Anything it cannot
// use falls back to the current month rather than to every month: a stale
// bookmark or a hand-edited URL should leave the list where it was, and
// widening it to a whole history is the one fallback that would surprise.
func newTxnScope(param string) txnScope {
	if param == allMonths {
		return txnScope{Value: allMonths, Label: "All months", All: true, from: allTimeFrom, to: allTimeTo}
	}
	from, to := monthRangeFor(param)
	return txnScope{Value: from.Time.Format("2006-01"), Label: monthLabel(from.Time), from: from, to: to}
}

// scopeFromRequest reads the scope the page that issued this request was
// showing, for the mutation handlers -- the same reason monthRangeFromRequest
// exists, and read out of the same HX-Current-URL header.
func scopeFromRequest(r *http.Request) txnScope {
	return newTxnScope(monthParamFromRequest(r))
}

// Bounds is the half-open [from, to) window the queries take.
func (s txnScope) Bounds() (from, to pgtype.Date) { return s.from, s.to }

// LabelLower is the bare month name the month-scoped empty states read
// ("No transactions in august"). It is meaningless for the all-time scope,
// which those templates branch away from before reaching it.
func (s txnScope) LabelLower() string { return monthLabelLower(s.from.Time) }

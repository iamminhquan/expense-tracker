package format

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// DateShort formats a DATE column as "11 Aug" -- used in the transaction
// list row and mobile card, where the year is implied by the month filter.
func DateShort(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02 Jan")
}

// DateLong formats a DATE column as "11 Aug 2026", for the all-months list,
// where the year DateShort leaves implied is not implied by anything.
func DateLong(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02 Jan 2006")
}

// Timestamp formats a TIMESTAMPTZ in loc, matching the "11 Aug 2026"
// convention the rest of the app uses for dates that need a year. The
// session list is the caller: a session can outlive the month it started in,
// unlike the transaction rows DateShort formats.
//
// The location is a parameter rather than the app's own because nothing else
// in this package needs a clock, and a formatter that hides which timezone it
// printed in is the kind that quietly disagrees with the rest of the page.
func Timestamp(t pgtype.Timestamptz, loc *time.Location) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(loc).Format("02 Jan 2006, 15:04")
}

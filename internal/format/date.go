package format

import "github.com/jackc/pgx/v5/pgtype"

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

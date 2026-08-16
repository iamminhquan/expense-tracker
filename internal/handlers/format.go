package handlers

import (
	"html/template"
	"strconv"
	"strings"

	"expensetracker/internal/i18n"

	"github.com/jackc/pgx/v5/pgtype"
)

// TemplateFuncs returns the FuncMap every page template needs for
// formatting money and dates per SPEC.md section 1: thousands-dot-separated
// integers with a trailing ₫, and dd/mm/yyyy dates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"vnd":        vnd,
		"vndSigned":  vndSigned,
		"vndBalance": vndBalance,
		"dateFull":   dateFull,
		"dateShort":  dateShort,
		"catName":    i18n.CategoryName,
		"swatches":   func() []string { return categorySwatches },
	}
}

// vnd formats the magnitude of n as thousands-dot-separated đồng, e.g.
// 50000 -> "50.000₫". The sign is never shown here; callers needing a sign
// use vndSigned (transaction rows) or vndBalance (a total that can itself
// be negative).
func vnd(n int64) string {
	if n < 0 {
		n = -n
	}
	return formatThousands(n) + "₫"
}

// vndSigned formats a transaction amount with the sign SPEC.md section 3.3
// assigns by transaction type: "-" for expense, "+" for anything else.
func vndSigned(n int64, txnType string) string {
	sign := "+"
	if txnType == "expense" {
		sign = "-"
	}
	return sign + vnd(n)
}

// vndBalance formats a total (e.g. "remaining this month") whose sign comes
// from the number's own sign, since unlike a single transaction it can
// itself be negative.
func vndBalance(n int64) string {
	sign := "+"
	if n < 0 {
		sign = "-"
	}
	return sign + vnd(n)
}

func formatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ".")
}

// dateFull formats a DATE column as dd/mm/yyyy, e.g. "11/08/2026" -- used in
// forms and the desktop transaction list's date column.
func dateFull(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02/01/2006")
}

// dateShort formats a DATE column as dd/mm, e.g. "11/08" -- used in the
// transaction list row and mobile card.
func dateShort(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02/01")
}

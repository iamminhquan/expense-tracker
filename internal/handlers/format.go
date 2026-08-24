package handlers

import (
	"html/template"
	"strconv"
	"strings"

	"expensetracker/internal/i18n"

	"github.com/jackc/pgx/v5/pgtype"
)

// TemplateFuncs returns the FuncMap every page template needs for
// formatting money and dates. The app was originally specified in
// Vietnamese convention (dots for thousands, dd/mm/yyyy); with the UI in
// English the rules are commas for thousands, a trailing ₫, and a
// spelled-out month ("11 Aug 2026").
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"vnd":        vnd,
		"vndSigned":  vndSigned,
		"vndBalance": vndBalance,
		"dateShort":  dateShort,
		"catName":    i18n.CategoryName,
		"countOf":    countOf,
		"swatches":   func() []string { return categorySwatches },
	}
}

// vnd formats the magnitude of n as comma-separated đồng, e.g.
// 50000 -> "50,000₫". The sign is never shown here; callers needing a sign
// use vndSigned (transaction rows) or vndBalance (a total that can itself
// be negative).
func vnd(n int64) string {
	if n < 0 {
		n = -n
	}
	return formatThousands(n) + "₫"
}

// vndSigned formats a transaction amount with the sign its type implies:
// "-" for expense, "+" for anything else.
func vndSigned(n int64, txnType string) string {
	sign := "+"
	if txnType == "expense" {
		sign = "-"
	}
	return sign + vnd(n)
}

// vndBalance formats the running balance, whose sign comes from the number
// itself rather than from a transaction type. Only the minus is printed: the
// balance is a standing amount, not a change, so a leading "+" would read as
// "went up by this much" rather than "this is what you have".
func vndBalance(n int64) string {
	if n < 0 {
		return "-" + vnd(n)
	}
	return vnd(n)
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
	return strings.Join(parts, ",")
}

// dateShort formats a DATE column as "11 Aug" -- used in the transaction
// list row and mobile card, where the year is implied by the month filter.
func dateShort(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02 Jan")
}

// countOf renders a count with its noun in agreement: "1 transaction",
// "2 transactions". English needs this and Vietnamese does not, so the
// templates carried a bare plural noun until the UI was translated.
//
// The count arrives as an int from len() in one template and as an int64
// straight out of a COUNT(*) in another, so it is taken as any rather than
// forcing one of the two call sites to convert in the template.
func countOf(n any, singular string) string {
	var count int64
	switch v := n.(type) {
	case int:
		count = int64(v)
	case int64:
		count = v
	case int32:
		count = int64(v)
	}
	if count == 1 {
		return "1 " + singular
	}
	return strconv.FormatInt(count, 10) + " " + singular + "s"
}

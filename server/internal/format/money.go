// Package format renders the app's values as the strings its pages show:
// money, dates, counts, and the handful of other display strings that are
// decided in Go rather than in a template.
//
// Nothing here reads a request or a database row; the handlers hand it
// finished values. That is what keeps the money and date conventions in one
// place and testable without either.
package format

import "strconv"

// VND formats the magnitude of n as comma-separated đồng, e.g.
// 50000 -> "50,000₫". The sign is never shown here; callers needing a sign
// use VNDSigned (transaction rows) or VNDBalance (a total that can itself
// be negative).
//
// The app was originally specified in Vietnamese convention (dots for
// thousands); with the UI in English the rule is commas and a trailing ₫.
func VND(n int64) string {
	if n < 0 {
		n = -n
	}
	return thousands(n) + "₫"
}

// VNDSigned formats a transaction amount with the sign its type implies:
// "-" for expense, "+" for anything else.
func VNDSigned(n int64, txnType string) string {
	sign := "+"
	if txnType == "expense" {
		sign = "-"
	}
	return sign + VND(n)
}

// VNDBalance formats the running balance, whose sign comes from the number
// itself rather than from a transaction type. Only the minus is printed: the
// balance is a standing amount, not a change, so a leading "+" would read as
// "went up by this much" rather than "this is what you have".
func VNDBalance(n int64) string {
	if n < 0 {
		return "-" + VND(n)
	}
	return VND(n)
}

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

package csvimport

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// NoColumn is the Mapping value for a role the file has no column for.
const NoColumn = -1

// Mapping says how to read one file: which column plays which role, and the
// two conventions that cannot be read off a single value -- what order the
// date parts are in, and whether a minus sign is what marks an expense.
//
// It travels through the browser as form fields rather than living on the
// server between steps, so every field here has to survive a round trip as
// text.
type Mapping struct {
	Date     int
	Amount   int
	Category int
	Note     int
	Type     int

	DateLayout        string
	NegativeIsExpense bool

	// FallbackCategory is the category name every row is filed under when
	// the file has no category column at all.
	FallbackCategory string
}

// ExportMapping returns the column arrangement for files this app exports.
func ExportMapping() Mapping {
	return Mapping{Date: 0, Type: 1, Category: 2, Amount: 3, Note: 4, DateLayout: DateISO}
}

// DateISO identifies the ISO 8601 date format (YYYY-MM-DD).
const DateISO = "iso"

// DateDMYSlash identifies the day-first slash-separated format (DD/MM/YYYY).
const DateDMYSlash = "dmy-slash"

// DateMDYSlash identifies the month-first slash-separated format (MM/DD/YYYY).
const DateMDYSlash = "mdy-slash"

// DateDMYDash identifies the day-first dash-separated format (DD-MM-YYYY).
const DateDMYDash = "dmy-dash"

// DateISOTime identifies the ISO 8601 datetime format (YYYY-MM-DDTHH:MM:SS).
const DateISOTime = "iso-datetime"

// DateFormat is one entry in the date-order picker.
type DateFormat struct {
	Key    string
	Label  string
	layout string
}

// DateFormats lists every date order the importer reads, in the order the
// picker offers them.
var DateFormats = []DateFormat{
	{DateISO, "2026-08-11 · YYYY-MM-DD", "2006-01-02"},
	{DateDMYSlash, "11/08/2026 · DD/MM/YYYY", "02/01/2006"},
	{DateMDYSlash, "08/11/2026 · MM/DD/YYYY", "01/02/2006"},
	{DateDMYDash, "11-08-2026 · DD-MM-YYYY", "02-01-2006"},
	{DateISOTime, "2026-08-11T09:30:00 · date and time", "2006-01-02T15:04:05"},
}

// ValidDateLayout reports whether key names a date layout the picker offers.
func ValidDateLayout(key string) bool {
	for _, f := range DateFormats {
		if f.Key == key {
			return true
		}
	}
	return false
}

// parseDate reads one value under the named layout. The date-and-time entry
// accepts an offset or a trailing Z as well, since an exporter that writes
// timestamps rarely writes only the bare kind.
func parseDate(value, key string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, f := range DateFormats {
		if f.Key != key {
			continue
		}
		if parsed, err := time.Parse(f.layout, value); err == nil {
			return truncateDay(parsed), true
		}
		if key == DateISOTime {
			if parsed, err := time.Parse(time.RFC3339, value); err == nil {
				return truncateDay(parsed), true
			}
			if date, _, found := strings.Cut(value, "T"); found {
				if parsed, err := time.Parse("2006-01-02", date); err == nil {
					return parsed, true
				}
			}
		}
		return time.Time{}, false
	}
	return time.Time{}, false
}

// truncateDay drops the clock from a timestamp: transactions carry a date,
// and keeping 09:30 would only make a row's month depend on a timezone.
func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// parseAmount reads a money value written the way people and exporters
// write it, and reports whether it could. The result keeps its sign; what a
// negative means is the Mapping's business, not this function's.
//
// Rounding to whole đồng is deliberate: amounts are stored as integers, and
// a file in a currency with cents would otherwise import nothing at all.
func parseAmount(value string) (amount int64, rounded, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, false
	}

	negative := false
	// Accounting notation: (45,000) is what a spreadsheet writes for -45000.
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		negative = true
		value = value[1 : len(value)-1]
	}

	var digits strings.Builder
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r == '.', r == ',':
			digits.WriteRune(r)
		case r == '-':
			negative = !negative
		case r == '+':
		default:
			// Currency symbols, spaces, non-breaking spaces and any other
			// decoration are dropped rather than rejected: "45 000 ₫" is one
			// number to everyone who is not a parser.
		}
	}
	cleaned := separatorsResolved(digits.String())
	if cleaned == "" {
		return 0, false, false
	}
	parsed, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, false, false
	}
	if negative {
		parsed = -parsed
	}
	whole := math.Round(parsed)
	return int64(whole), whole != parsed, true
}

// separatorsResolved decides which of . and , is the decimal point and
// removes the other.
//
// The hard case is a single separator followed by exactly three digits:
// "45.000" is forty-five thousand in half the world and forty-five in the
// other half. It is read as a thousands separator, because three decimal
// places in a money column is rare and this app's own currency has none.
func separatorsResolved(s string) string {
	lastDot, lastComma := strings.LastIndex(s, "."), strings.LastIndex(s, ",")
	switch {
	case lastDot >= 0 && lastComma >= 0:
		// Whichever comes last is the decimal point.
		decimal, thousands := ".", ","
		if lastComma > lastDot {
			decimal, thousands = ",", "."
		}
		return strings.Replace(strings.ReplaceAll(s, thousands, ""), decimal, ".", 1)
	case lastDot >= 0 || lastComma >= 0:
		sep := "."
		at := lastDot
		if lastComma >= 0 {
			sep, at = ",", lastComma
		}
		if strings.Count(s, sep) > 1 || len(s)-at-1 == 3 {
			return strings.ReplaceAll(s, sep, "")
		}
		return strings.Replace(s, sep, ".", 1)
	}
	return s
}

// typeWords maps what a type column can say to what the app stores. The
// lists are exact matches rather than prefixes: "in" means income, but
// "invoice" does not.
var typeWords = map[string]string{
	"expense": "expense", "expenses": "expense", "chi": "expense", "chi tieu": "expense",
	"chi tiêu": "expense", "debit": "expense", "withdrawal": "expense", "payment": "expense",
	"out": "expense", "-": "expense",

	"income": "income", "thu": "income", "thu nhap": "income", "thu nhập": "income",
	"credit": "income", "deposit": "income", "in": "income", "+": "income",
}

// transactionType reads one value of a type column.
func transactionType(value string) (string, bool) {
	t, ok := typeWords[normalize(value)]
	return t, ok
}

// normalize is how every header and type word is compared: trimmed,
// lowercased, inner runs of whitespace collapsed. Tones are left alone --
// the alias lists spell out both the toned and untoned forms, which reads
// better than carrying a Unicode normaliser to strip them.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

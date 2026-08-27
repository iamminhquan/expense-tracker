package csvimport

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// sampleRows is how many rows the mapping screen shows back to the user.
// It is a reminder of what the columns hold, not a preview of the import.
const sampleRows = 5

// contentShare is how much of a column has to look like a date, an amount
// or a type word before that column is guessed into the role. It is short
// of 1 so a few blank or broken cells cannot veto an otherwise obvious
// column.
const contentShare = 0.8

// Sheet is what a file looks like before anyone has said how to read it:
// its columns, a few rows to look at, and the best guess at what each
// column means.
type Sheet struct {
	Columns []string
	Sample  [][]string
	Rows    int
	Guess   Mapping

	// Exact marks a file this app exported. Its mapping is known, so the
	// mapping screen is skipped entirely and the round trip stays as short
	// as it was before other formats were accepted.
	Exact bool

	// AmbiguousDate marks a date column where every value fits both
	// day-first and month-first. The guess is day-first, and this is what
	// tells the screen to say so out loud -- it is the only wrong guess in
	// the importer that still produces rows that look right.
	AmbiguousDate bool

	Fingerprint string
}

// Sniff reads a file and proposes how to import it.
//
// It reads the whole file rather than a prefix: the fingerprint has to
// cover every byte, and guessing the date order means finding the one row
// in two hundred whose day is past the 12th.
func Sniff(r io.Reader) (*Sheet, error) {
	digest := sha256.New()
	reader := csv.NewReader(io.TeeReader(r, digest))
	// A foreign file may well have ragged rows; that is the row's problem
	// at import time, not a reason to refuse to look at the file.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	header := records[0]
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	rows := records[1:]
	if len(rows) > MaxRows {
		return nil, ErrTooManyRows
	}

	sheet := &Sheet{
		Columns:     header,
		Sample:      rows[:min(len(rows), sampleRows)],
		Rows:        len(rows),
		Fingerprint: hex.EncodeToString(digest.Sum(nil)),
	}
	if isExportHeader(header) {
		sheet.Exact = true
		sheet.Guess = ExportMapping()
		return sheet, nil
	}
	sheet.Guess, sheet.AmbiguousDate = guessMapping(header, rows)
	return sheet, nil
}

// isExportHeader reports whether this is the header the app's own export
// writes, which is what lets a round trip skip the mapping screen.
func isExportHeader(header []string) bool {
	if len(header) != len(columns) {
		return false
	}
	for i, want := range columns {
		got := normalize(header[i])
		if got != want && !(want == "note" && got == "description") {
			return false
		}
	}
	return true
}

// guessMapping proposes a role for each column, by name first and by
// content second. Every guess is rendered into a control the user can
// change, so being wrong costs a click -- which is why the content rules
// below are allowed to be as rough as they are.
func guessMapping(header []string, rows [][]string) (Mapping, bool) {
	m := Mapping{Date: NoColumn, Amount: NoColumn, Category: NoColumn, Note: NoColumn, Type: NoColumn}
	taken := make([]bool, len(header))

	for i, name := range header {
		role, ok := roleForName(normalize(name))
		if !ok || m.column(role) != NoColumn {
			continue
		}
		m.setColumn(role, i)
		taken[i] = true
	}

	if m.Date == NoColumn {
		if i, ok := columnLooking(rows, taken, len(header), looksLikeDate); ok {
			m.Date, taken[i] = i, true
		}
	}
	if m.Amount == NoColumn {
		if i, ok := columnLooking(rows, taken, len(header), looksLikeAmount); ok {
			m.Amount, taken[i] = i, true
		}
	}
	if m.Type == NoColumn {
		if i, ok := columnLooking(rows, taken, len(header), looksLikeType); ok {
			m.Type, taken[i] = i, true
		}
	}
	// Of the columns nobody claimed, the one that repeats itself most is
	// the category and the one that repeats least is the note. A category
	// is drawn from a short list by definition; a note is written fresh
	// each time.
	if m.Category == NoColumn || m.Note == NoColumn {
		byVariety := remainingByVariety(rows, taken, len(header))
		if m.Category == NoColumn && len(byVariety) > 0 {
			m.Category, taken[byVariety[0]] = byVariety[0], true
			byVariety = byVariety[1:]
		}
		if m.Note == NoColumn && len(byVariety) > 0 {
			m.Note = byVariety[len(byVariety)-1]
		}
	}

	ambiguous := false
	if m.Date != NoColumn {
		m.DateLayout, ambiguous = guessDateLayout(columnValues(rows, m.Date))
	} else {
		m.DateLayout = DateISO
	}
	if m.Amount != NoColumn {
		m.NegativeIsExpense = anyNegative(columnValues(rows, m.Amount))
	}
	return m, ambiguous
}

// headerAliases is what each role can be called. Both the toned and the
// untoned Vietnamese spellings are listed, because an exporter writing to
// CSV may have dropped the tones and a lookup table is easier to read (and
// to extend when a real file turns up) than a normaliser that strips them.
var headerAliases = map[string][]string{
	"date": {
		"date", "transaction date", "trans date", "posting date", "day", "time",
		"ngày", "ngay", "ngày giao dịch", "ngay giao dich", "thời gian", "thoi gian",
	},
	"amount": {
		"amount", "value", "sum", "total", "money", "price",
		"số tiền", "so tien", "giá trị", "gia tri", "tiền", "tien",
	},
	"category": {
		"category", "categories", "group", "hạng mục", "hang muc",
		"danh mục", "danh muc", "nhóm", "nhom", "loại chi tiêu", "loai chi tieu",
	},
	"note": {
		"note", "notes", "description", "memo", "detail", "details", "remark", "remarks",
		"ghi chú", "ghi chu", "nội dung", "noi dung", "diễn giải", "dien giai", "mô tả", "mo ta",
	},
	"type": {
		"type", "kind", "direction", "loại", "loai", "loại giao dịch", "loai giao dich",
		"thu chi", "thu/chi", "in/out",
	},
}

func roleForName(name string) (string, bool) {
	for role, aliases := range headerAliases {
		for _, alias := range aliases {
			if name == alias {
				return role, true
			}
		}
	}
	return "", false
}

func (m Mapping) column(role string) int {
	switch role {
	case "date":
		return m.Date
	case "amount":
		return m.Amount
	case "category":
		return m.Category
	case "note":
		return m.Note
	case "type":
		return m.Type
	}
	return NoColumn
}

func (m *Mapping) setColumn(role string, i int) {
	switch role {
	case "date":
		m.Date = i
	case "amount":
		m.Amount = i
	case "category":
		m.Category = i
	case "note":
		m.Note = i
	case "type":
		m.Type = i
	}
}

func columnValues(rows [][]string, i int) []string {
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if i < len(row) {
			values = append(values, row[i])
		}
	}
	return values
}

// columnLooking returns the first unclaimed column where enough of the
// values satisfy looks.
func columnLooking(rows [][]string, taken []bool, width int, looks func(string) bool) (int, bool) {
	for i := 0; i < width; i++ {
		if taken[i] {
			continue
		}
		values := columnValues(rows, i)
		matched, counted := 0, 0
		for _, v := range values {
			if strings.TrimSpace(v) == "" {
				continue
			}
			counted++
			if looks(v) {
				matched++
			}
		}
		if counted > 0 && float64(matched)/float64(counted) >= contentShare {
			return i, true
		}
	}
	return 0, false
}

// remainingByVariety orders the unclaimed columns from fewest distinct
// values to most.
func remainingByVariety(rows [][]string, taken []bool, width int) []int {
	var remaining []int
	variety := map[int]int{}
	for i := 0; i < width; i++ {
		if taken[i] {
			continue
		}
		distinct := map[string]bool{}
		for _, v := range columnValues(rows, i) {
			distinct[normalize(v)] = true
		}
		remaining = append(remaining, i)
		variety[i] = len(distinct)
	}
	for a := 1; a < len(remaining); a++ {
		for b := a; b > 0 && variety[remaining[b]] < variety[remaining[b-1]]; b-- {
			remaining[b], remaining[b-1] = remaining[b-1], remaining[b]
		}
	}
	return remaining
}

func looksLikeDate(v string) bool {
	for _, f := range DateFormats {
		if _, ok := parseDate(v, f.Key); ok {
			return true
		}
	}
	return false
}

func looksLikeAmount(v string) bool {
	_, _, ok := parseAmount(v)
	return ok
}

func looksLikeType(v string) bool {
	_, ok := transactionType(v)
	return ok
}

// guessDateLayout picks the order that reads the most of the column, and
// reports whether day-first and month-first are equally good -- which they
// are for any column whose days never pass the 12th.
func guessDateLayout(values []string) (string, bool) {
	best, bestScore := DateISO, -1
	score := map[string]int{}
	for _, f := range DateFormats {
		for _, v := range values {
			if strings.TrimSpace(v) == "" {
				continue
			}
			if _, ok := parseDate(v, f.Key); ok {
				score[f.Key]++
			}
		}
		if score[f.Key] > bestScore {
			best, bestScore = f.Key, score[f.Key]
		}
	}
	ambiguous := bestScore > 0 && score[DateDMYSlash] == score[DateMDYSlash] && score[DateDMYSlash] == bestScore
	if ambiguous {
		best = DateDMYSlash
	}
	return best, ambiguous
}

func anyNegative(values []string) bool {
	for _, v := range values {
		if amount, _, ok := parseAmount(v); ok && amount < 0 {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

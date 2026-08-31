// Package csvimport turns the CSV the app exports back into transactions
// an account can be given.
//
// It reads and validates; it never touches the database. Everything it
// needs to know about the account arrives as a catalog of categories, and
// everything it decides leaves as an Import for the caller to apply. That
// boundary is what lets the whole format -- every column, every rule, every
// message -- be tested without Postgres.
package csvimport

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/txnrule"
)

// Category is one category the account can already spend against. Slug is
// empty for a category the user created; a shared default carries one, and
// it -- not the name column -- is what identifies it.
type Category struct {
	ID   int64
	Name string
	Slug string
	Type string
}

// Row is one transaction the file asks for. CategoryID is 0 when the
// category does not exist yet, in which case CategoryName names the
// NewCategory the caller has to create first.
type Row struct {
	Line         int
	Date         time.Time
	Type         string
	Amount       int64
	Note         string
	CategoryID   int64
	CategoryName string
}

// NewCategory is a category the file mentions that the account does not
// have. It carries no color: the palette is the application's, and this
// package has no business holding a second copy of it.
type NewCategory struct {
	Name string
	Type string
}

// RowError is one rejected line, named by the line number the user sees in
// their spreadsheet -- the header is line 1.
type RowError struct {
	Line    int
	Message string
}

// Import is the plan a file amounts to. Fingerprint is a digest of the
// bytes it was read from: the preview shows a plan, the confirm re-uploads
// the file, and comparing fingerprints is what proves the second upload is
// the file the first one described.
type Import struct {
	Rows          []Row
	NewCategories []NewCategory
	Errors        []RowError
	Fingerprint   string

	// Rounded counts the rows whose amount had a fractional part. VND has
	// no minor unit, so a file in a currency that does still imports -- the
	// preview says how many rows were rounded rather than the app silently
	// changing numbers.
	Rounded int
}

// MaxRows caps a single import at 2000 transactions.
const MaxRows = 2000

// ErrTooManyRows reports a file with more than MaxRows data rows.
var ErrTooManyRows = fmt.Errorf("file has more than %d rows", MaxRows)

// columns is the header the export writes, and the only one accepted.
var columns = []string{"date", "type", "category", "amount", "note"}

// Plan reads the file the way the Mapping says to, and works out what
// importing it would do. An unreadable file is an error; a bad line is not
// -- it lands in Errors so the caller can show every problem at once
// instead of one per upload.
func Plan(r io.Reader, m Mapping, catalog []Category, now time.Time) (*Import, error) {
	digest := sha256.New()
	reader := csv.NewReader(io.TeeReader(r, digest))
	// Ragged rows are a problem for the rows that are ragged, not for the
	// file: a short line is reported by line number like any other bad one.
	reader.FieldsPerRecord = -1

	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	index := newCategoryIndex(catalog)
	imp := &Import{}
	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			imp.Errors = append(imp.Errors, RowError{Line: line, Message: "this line could not be read"})
			continue
		}
		if line-1 > MaxRows {
			return nil, ErrTooManyRows
		}
		row, rounded, rowErr := parseRow(record, line, m, now)
		if rowErr != "" {
			imp.Errors = append(imp.Errors, RowError{Line: line, Message: rowErr})
			continue
		}
		if rounded {
			imp.Rounded++
		}
		row.CategoryID = index.lookup(row.CategoryName, row.Type)
		if row.CategoryID == 0 {
			index.plan(imp, row.CategoryName, row.Type)
		}
		imp.Rows = append(imp.Rows, row)
	}
	imp.Fingerprint = hex.EncodeToString(digest.Sum(nil))
	return imp, nil
}

// parseRow reads one line under the mapping. It returns the row, whether
// the amount had to be rounded to whole đồng, and an empty message on
// success.
//
// The date, note and future-date rules are the quick-add form's, shared
// through internal/txnrule rather than restated: a row this package accepts
// must be one the form would have accepted too, or the app grows a second,
// laxer way in.
func parseRow(record []string, line int, m Mapping, now time.Time) (Row, bool, string) {
	if width := m.widest(); width >= len(record) {
		return Row{}, false, fmt.Sprintf("this line has %d columns, and the mapping needs at least %d", len(record), width+1)
	}

	date, ok := parseDate(record[m.Date], m.DateLayout)
	if !ok {
		return Row{}, false, fmt.Sprintf("date %q is not written as %s", record[m.Date], layoutLabel(m.DateLayout))
	}
	if txnrule.TooFarInFuture(date, now) {
		return Row{}, false, fmt.Sprintf("date %s is more than %d days in the future", record[m.Date], txnrule.MaxFutureDays)
	}

	signed, rounded, ok := parseAmount(record[m.Amount])
	if !ok {
		return Row{}, false, fmt.Sprintf("amount %q is not a number", record[m.Amount])
	}
	amount := signed
	if amount < 0 {
		amount = -amount
	}
	if amount == 0 {
		return Row{}, false, "amount is zero"
	}

	txnType, msg := rowType(record, m, signed)
	if msg != "" {
		return Row{}, false, msg
	}

	category := m.FallbackCategory
	if m.Category != NoColumn {
		category = strings.TrimSpace(record[m.Category])
	}
	if category == "" {
		return Row{}, false, "this line has no category, and no fallback category was chosen"
	}

	note := ""
	if m.Note != NoColumn {
		note = record[m.Note]
	}
	if txnrule.NoteTooLong(note) {
		return Row{}, false, fmt.Sprintf("note is longer than %d characters", txnrule.MaxNoteRunes)
	}

	return Row{
		Line: line, Date: date, Type: txnType, Amount: amount,
		Note: note, CategoryName: category,
	}, rounded, ""
}

// rowType decides whether a line is money in or money out. A type column
// answers it outright; without one, the sign does, and a file with neither
// is read as spending -- which is what a file handed to an expense tracker
// almost always is.
func rowType(record []string, m Mapping, signed int64) (string, string) {
	if m.Type != NoColumn {
		txnType, ok := transactionType(record[m.Type])
		if !ok {
			return "", fmt.Sprintf("type %q is not a word this recognises (expense/income, chi/thu, debit/credit)", record[m.Type])
		}
		return txnType, ""
	}
	if m.NegativeIsExpense {
		if signed < 0 {
			return "expense", ""
		}
		return "income", ""
	}
	return "expense", ""
}

// widest is the highest column index the mapping reads, so a short line can
// be reported as short rather than as a date that would not parse.
func (m Mapping) widest() int {
	widest := m.Date
	for _, i := range []int{m.Amount, m.Category, m.Note, m.Type} {
		if i > widest {
			widest = i
		}
	}
	return widest
}

func layoutLabel(key string) string {
	for _, f := range DateFormats {
		if f.Key == key {
			return f.Label
		}
	}
	return key
}

// categoryIndex answers "which category does this name mean" for one
// account. A default is found through its slug's label from i18n rather
// than through the name stored beside it, because the slug is what
// identifies a default -- the stored name is only a fallback label.
type categoryIndex map[string]int64

func newCategoryIndex(catalog []Category) categoryIndex {
	index := make(categoryIndex, len(catalog))
	for _, c := range catalog {
		name := c.Name
		if c.Slug != "" {
			if label := i18n.NameForSlug(c.Slug); label != "" {
				name = label
			}
		}
		index[MatchKey(name, c.Type)] = c.ID
	}
	return index
}

func (i categoryIndex) lookup(name, txnType string) int64 {
	return i[MatchKey(name, txnType)]
}

// plan records a category the file needs and the account lacks, once. It
// marks the key as planned so a name appearing on twenty lines is created
// once, and so the entry keeps the spelling of its first appearance.
func (i categoryIndex) plan(imp *Import, name, txnType string) {
	key := MatchKey(name, txnType)
	if _, planned := i[key]; planned {
		return
	}
	i[key] = 0
	imp.NewCategories = append(imp.NewCategories, NewCategory{Name: name, Type: txnType})
}

// MatchKey is the rule for when two category names mean the same category:
// same type, same name ignoring case. It is exported because the caller
// applying an Import has to pair each Row with the NewCategory it will be
// given, and a second, subtly different rule there would leave rows pointing
// at a category that was never created.
func MatchKey(name, txnType string) string {
	return txnType + "\x00" + strings.ToLower(name)
}

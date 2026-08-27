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
	"strconv"
	"strings"
	"time"

	"expensetracker/internal/i18n"
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
}

// MaxRows caps a single import. The whole batch is inserted in one
// transaction and rendered in one preview, and neither wants an unbounded
// file behind it.
const MaxRows = 2000

// ErrTooManyRows reports a file with more than MaxRows data rows.
var ErrTooManyRows = fmt.Errorf("file has more than %d rows", MaxRows)

// columns is the header the export writes, and the only one accepted.
var columns = []string{"date", "type", "category", "amount", "note"}

// dateLayout is the one date format read, because it is the one the export
// writes. Accepting more would mean guessing between 03/04 as March 4th and
// April 3rd, and guessing wrong silently.
const dateLayout = "2006-01-02"

// futureDays and maxNote mirror the limits handleCreateTransaction applies
// to the quick-add form.
const (
	futureDays = 7
	maxNote    = 200
)

// Plan reads the file and works out what importing it would do. A malformed
// header or an unreadable file is an error; a bad line is not -- it lands in
// Errors so the caller can show every problem at once instead of one per
// upload.
func Plan(r io.Reader, catalog []Category, now time.Time) (*Import, error) {
	digest := sha256.New()
	reader := csv.NewReader(io.TeeReader(r, digest))
	reader.FieldsPerRecord = len(columns)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := checkHeader(header); err != nil {
		return nil, err
	}

	index := newCategoryIndex(catalog)
	imp := &Import{}
	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			imp.Errors = append(imp.Errors, RowError{Line: line, Message: "this line does not have the 5 expected columns"})
			continue
		}
		if line-1 > MaxRows {
			return nil, ErrTooManyRows
		}
		row, rowErr := parseRow(record, line, now)
		if rowErr != "" {
			imp.Errors = append(imp.Errors, RowError{Line: line, Message: rowErr})
			continue
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

// checkHeader accepts the export's own header, case-insensitively and with
// the BOM the export writes ahead of it, plus "description" for "note" --
// the column is called Note in the file and description in the database,
// and a hand-edited file may well say either.
func checkHeader(header []string) error {
	if len(header) != len(columns) {
		return fmt.Errorf("expected %d columns (%s), got %d", len(columns), strings.Join(columns, ", "), len(header))
	}
	for i, want := range columns {
		got := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(header[i], "\ufeff")))
		if got == want || (want == "note" && got == "description") {
			continue
		}
		return fmt.Errorf("column %d is %q, expected %q", i+1, header[i], want)
	}
	return nil
}

func parseRow(record []string, line int, now time.Time) (Row, string) {
	date, err := time.Parse(dateLayout, strings.TrimSpace(record[0]))
	if err != nil {
		return Row{}, fmt.Sprintf("date %q is not in YYYY-MM-DD form", record[0])
	}
	txnType := strings.ToLower(strings.TrimSpace(record[1]))
	if txnType != "expense" && txnType != "income" {
		return Row{}, fmt.Sprintf("type %q must be expense or income", record[1])
	}
	amount, err := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
	if err != nil || amount <= 0 {
		return Row{}, fmt.Sprintf("amount %q must be a whole number above zero, with no separators", record[3])
	}
	category := strings.TrimSpace(record[2])
	if category == "" {
		return Row{}, "category is empty"
	}
	// The two rules below are the quick-add form's, repeated rather than
	// relaxed: a row this package accepts must be one the form would have
	// accepted too, or the app grows a second, laxer way in.
	if date.After(now.AddDate(0, 0, futureDays)) {
		return Row{}, fmt.Sprintf("date %s is more than %d days in the future", record[0], futureDays)
	}
	if len([]rune(record[4])) > maxNote {
		return Row{}, fmt.Sprintf("note is longer than %d characters", maxNote)
	}
	return Row{
		Line: line, Date: date, Type: txnType, Amount: amount,
		Note: record[4], CategoryName: category,
	}, ""
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

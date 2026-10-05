package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"expensetracker/internal/csvimport"
	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/gin-gonic/gin"
)

// importMaxBytes mirrors handlers.importMaxBytes: a file this size is
// already far past csvimport.MaxRows, so the limit that actually bites is
// the row count -- this one exists only to refuse a multi-gigabyte body
// before it's read into memory.
const importMaxBytes = 1 << 20

// mappingFields mirrors handlers/import_mapping.go's table exactly, field
// names included -- the client's form field names and these have to agree
// the same way the HTML template's did.
var mappingFields = []struct {
	Label string
	Field string
	get   func(*csvimport.Mapping) *int
}{
	{"Date", "date_col", func(m *csvimport.Mapping) *int { return &m.Date }},
	{"Amount", "amount_col", func(m *csvimport.Mapping) *int { return &m.Amount }},
	{"Type", "type_col", func(m *csvimport.Mapping) *int { return &m.Type }},
	{"Category", "category_col", func(m *csvimport.Mapping) *int { return &m.Category }},
	{"Note", "note_col", func(m *csvimport.Mapping) *int { return &m.Note }},
}

// mappingNeededResponse is what the client gets back when the uploaded
// file isn't one $pend exported itself (sheet.Exact) and no mapping has
// been submitted yet -- the JSON equivalent of handlers.importMappingView,
// everything the mapping screen needs to draw itself and nothing it
// doesn't.
type mappingNeededResponse struct {
	NeedsMapping  bool            `json:"needsMapping"`
	Columns       []string        `json:"columns"`
	Sample        [][]string      `json:"sample"`
	Rows          int             `json:"rows"`
	Guess         mappingDTO      `json:"guess"`
	DateFormats   []dateFormatDTO `json:"dateFormats"`
	AmbiguousDate bool            `json:"ambiguousDate"`
	CategoryNames []string        `json:"categoryNames"`
	Fingerprint   string          `json:"fingerprint"`
}

// dateFormatDTO wraps csvimport.DateFormat, which has no JSON tags of its
// own (it was never meant to leave the HTML template that ranges over
// DateFormats directly) and would otherwise serialize its exported fields
// as "Key"/"Label" -- inconsistent with every other camelCase field this
// API returns. Key is what the client submits back as date_layout; the
// unexported layout (the actual time.Parse format string) stays
// server-side, which is exactly right -- the client never needs it.
type dateFormatDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func newDateFormatDTOs(formats []csvimport.DateFormat) []dateFormatDTO {
	out := make([]dateFormatDTO, len(formats))
	for i, f := range formats {
		out[i] = dateFormatDTO{Key: f.Key, Label: f.Label}
	}
	return out
}

// mappingDTO is csvimport.Mapping's column-role fields addressed by name
// instead of position, for a client that has no reason to know the field
// order mappingFields iterates in.
type mappingDTO struct {
	DateCol           int    `json:"dateCol"`
	AmountCol         int    `json:"amountCol"`
	TypeCol           int    `json:"typeCol"`
	CategoryCol       int    `json:"categoryCol"`
	NoteCol           int    `json:"noteCol"`
	DateLayout        string `json:"dateLayout"`
	NegativeIsExpense bool   `json:"negativeIsExpense"`
	FallbackCategory  string `json:"fallbackCategory"`
}

func newMappingDTO(m csvimport.Mapping) mappingDTO {
	return mappingDTO{
		DateCol: m.Date, AmountCol: m.Amount, TypeCol: m.Type, CategoryCol: m.Category, NoteCol: m.Note,
		DateLayout: m.DateLayout, NegativeIsExpense: m.NegativeIsExpense, FallbackCategory: m.FallbackCategory,
	}
}

// importPreviewResponse mirrors handlers.previewData's importView fields,
// minus Mapping (hidden form fields that existed only to round-trip
// through the browser's own form submission -- a JSON client already
// holds the mapping it sent and resends it with confirm=1 itself).
type importPreviewResponse struct {
	Preview       bool                   `json:"preview"`
	RowCount      int                    `json:"rowCount"`
	NewCategories []importNewCategoryDTO `json:"newCategories"`
	Errors        []rowErrorDTO          `json:"errors"`
	MoreErrors    int                    `json:"moreErrors"`
	Rounded       int                    `json:"rounded"`
	Duplicates    int                    `json:"duplicates"`
	Fingerprint   string                 `json:"fingerprint"`
	Importable    bool                   `json:"importable"`
	DateSuspect   bool                   `json:"dateSuspect"`
}

// importNewCategoryDTO and rowErrorDTO wrap csvimport.NewCategory/RowError,
// which -- like csvimport.DateFormat above -- have no JSON tags of their
// own and would otherwise serialize as "Name"/"Type" and "Line"/"Message"
// instead of this API's camelCase convention.
type importNewCategoryDTO struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type rowErrorDTO struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func newImportNewCategoryDTOs(categories []csvimport.NewCategory) []importNewCategoryDTO {
	out := make([]importNewCategoryDTO, len(categories))
	for i, c := range categories {
		out[i] = importNewCategoryDTO{Name: c.Name, Type: c.Type}
	}
	return out
}

func newRowErrorDTOs(errs []csvimport.RowError) []rowErrorDTO {
	out := make([]rowErrorDTO, len(errs))
	for i, e := range errs {
		out[i] = rowErrorDTO{Line: e.Line, Message: e.Message}
	}
	return out
}

// maxShownErrors mirrors handlers.maxShownErrors.
const maxShownErrors = 20

type importResultResponse struct {
	Imported int    `json:"imported"`
	Month    string `json:"month"`
}

// importTransactionsHandler mirrors handlers.importHandler's whole
// multi-step flow -- sniff, maybe ask for a mapping, preview, confirm --
// collapsed into one endpoint a client drives by what it sends: no
// mapping yet gets a mappingNeededResponse (unless the file is one $pend
// exported, in which case it's skipped entirely, same as the HTML side);
// a mapping without confirm=1 gets an importPreviewResponse; confirm=1
// with a matching fingerprint applies the plan and returns
// importResultResponse.
func importTransactionsHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxBytes)

		fileHeader, err := c.FormFile("file")
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "that file could not be read, choose a .csv file smaller than 1 MB")
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "that file could not be read, choose a .csv file smaller than 1 MB")
			return
		}
		defer file.Close()

		sheet, err := csvimport.Sniff(file)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, planFailureMessage(err))
			return
		}

		catalog, err := importCatalog(c.Request.Context(), deps, userID)
		if err != nil {
			log.Printf("import: load categories: %v", err)
			errorResponse(c, http.StatusInternalServerError, "could not load categories")
			return
		}

		mapping, submitted := mappingFromForm(c)
		switch {
		case !submitted && sheet.Exact:
			mapping = sheet.Guess
		case !submitted:
			names, err := categoryNamesForUser(c.Request.Context(), deps, userID)
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "could not load categories")
				return
			}
			c.JSON(http.StatusOK, mappingNeededResponse{
				NeedsMapping: true, Columns: sheet.Columns, Sample: sheet.Sample, Rows: sheet.Rows,
				Guess: newMappingDTO(sheet.Guess), DateFormats: newDateFormatDTOs(csvimport.DateFormats),
				AmbiguousDate: sheet.AmbiguousDate, CategoryNames: names, Fingerprint: sheet.Fingerprint,
			})
			return
		default:
			if msg := validateMapping(mapping, len(sheet.Columns)); msg != "" {
				errorResponse(c, http.StatusBadRequest, msg)
				return
			}
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			log.Printf("import: rewind upload: %v", err)
			errorResponse(c, http.StatusInternalServerError, "could not read the file")
			return
		}
		plan, err := csvimport.Plan(file, mapping, catalog, time.Now().In(vietnamLocation))
		if err != nil {
			errorResponse(c, http.StatusBadRequest, planFailureMessage(err))
			return
		}

		duplicates, err := countImportDuplicates(c.Request.Context(), deps, userID, plan)
		if err != nil {
			log.Printf("import: count duplicates: %v", err)
			errorResponse(c, http.StatusInternalServerError, "could not check for duplicates")
			return
		}

		if c.PostForm("confirm") == "" {
			shown, more := plan.Errors, 0
			if len(shown) > maxShownErrors {
				shown, more = shown[:maxShownErrors], len(plan.Errors)-maxShownErrors
			}
			// newImportNewCategoryDTOs/newRowErrorDTOs build via make([]T,
			// len(...)), which is never nil even for a 0-length input --
			// unlike csvimport.Plan's own Errors/NewCategories fields,
			// which go nil (not just empty) for a clean file or one that
			// names no new category. See .claude/rules/json-api-conventions.md.
			c.JSON(http.StatusOK, importPreviewResponse{
				Preview: true, RowCount: len(plan.Rows), NewCategories: newImportNewCategoryDTOs(plan.NewCategories),
				Errors: newRowErrorDTOs(shown), MoreErrors: more, Rounded: plan.Rounded, Duplicates: duplicates,
				Fingerprint: plan.Fingerprint, Importable: importable(plan), DateSuspect: mostlyDateErrors(plan.Errors),
			})
			return
		}

		if c.PostForm("fingerprint") != plan.Fingerprint {
			errorResponse(c, http.StatusConflict, "this is not the file you previewed, preview it again before importing")
			return
		}
		if !importable(plan) {
			errorResponse(c, http.StatusBadRequest, "this file still has lines that cannot be imported")
			return
		}

		if err := applyImport(c.Request.Context(), deps, userID, plan); err != nil {
			log.Printf("import: apply: %v", err)
			errorResponse(c, http.StatusInternalServerError, "the import could not be saved, nothing was changed, please try again")
			return
		}

		c.JSON(http.StatusOK, importResultResponse{Imported: len(plan.Rows), Month: latestMonth(plan)})
	}
}

// mappingFromForm mirrors handlers.mappingFromForm, reading from Gin's
// multipart form instead of net/http's directly.
func mappingFromForm(c *gin.Context) (csvimport.Mapping, bool) {
	if c.PostForm("mapped") == "" {
		return csvimport.Mapping{}, false
	}
	m := csvimport.Mapping{
		DateLayout:        c.PostForm("date_layout"),
		NegativeIsExpense: c.PostForm("negative_is_expense") != "",
		FallbackCategory:  strings.TrimSpace(c.PostForm("fallback_category")),
	}
	for _, f := range mappingFields {
		*f.get(&m) = columnIndex(c.PostForm(f.Field))
	}
	return m, true
}

func columnIndex(value string) int {
	i, err := strconv.Atoi(value)
	if err != nil || i < 0 {
		return csvimport.NoColumn
	}
	return i
}

// validateMapping mirrors handlers.validateMapping exactly.
func validateMapping(m csvimport.Mapping, columns int) string {
	if m.Date == csvimport.NoColumn {
		return "pick the column that holds the date"
	}
	if m.Amount == csvimport.NoColumn {
		return "pick the column that holds the amount"
	}
	if !csvimport.ValidDateLayout(m.DateLayout) {
		return "pick how the dates in this file are written"
	}
	if m.Category == csvimport.NoColumn && m.FallbackCategory == "" {
		return "this file has no category column, so pick a category to file everything under"
	}
	used := map[int]string{}
	for _, f := range mappingFields {
		i := *f.get(&m)
		if i == csvimport.NoColumn {
			continue
		}
		if i >= columns {
			return "that column is not in this file, pick again"
		}
		if already, taken := used[i]; taken {
			return "one column cannot be both " + already + " and " + f.Label
		}
		used[i] = f.Label
	}
	return ""
}

func mostlyDateErrors(errs []csvimport.RowError) bool {
	if len(errs) == 0 {
		return false
	}
	dates := 0
	for _, e := range errs {
		if strings.HasPrefix(e.Message, "date ") {
			dates++
		}
	}
	return dates*2 > len(errs)
}

func importable(plan *csvimport.Import) bool {
	return len(plan.Errors) == 0 && len(plan.Rows) > 0
}

func planFailureMessage(err error) string {
	if errors.Is(err, csvimport.ErrTooManyRows) {
		return fmt.Sprintf("this file has more than %d rows, split it and import the parts", csvimport.MaxRows)
	}
	return "this file is not shaped like the CSV $pend exports: " + err.Error()
}

func importCatalog(ctx context.Context, deps Deps, userID int64) ([]csvimport.Category, error) {
	rows, err := deps.Queries.ListCategoriesForUser(ctx, pgval.Int64(userID))
	if err != nil {
		return nil, err
	}
	catalog := make([]csvimport.Category, 0, len(rows))
	for _, c := range rows {
		catalog = append(catalog, csvimport.Category{ID: c.ID, Name: c.Name, Slug: c.Slug.String, Type: c.Type})
	}
	return catalog, nil
}

func categoryNamesForUser(ctx context.Context, deps Deps, userID int64) ([]string, error) {
	rows, err := deps.Queries.ListCategoriesForUser(ctx, pgval.Int64(userID))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, c := range rows {
		names = append(names, i18n.CategoryName(c.Slug, c.Name))
	}
	return names, nil
}

// countImportDuplicates mirrors handlers.countImportDuplicates exactly,
// reusing this package's own txnFilters (transaction_query.go) instead of
// handlers'.
func countImportDuplicates(ctx context.Context, deps Deps, userID int64, plan *csvimport.Import) (int, error) {
	if len(plan.Rows) == 0 {
		return 0, nil
	}
	first, last := plan.Rows[0].Date, plan.Rows[0].Date
	for _, row := range plan.Rows {
		if row.Date.Before(first) {
			first = row.Date
		}
		if row.Date.After(last) {
			last = row.Date
		}
	}

	existing, err := deps.Queries.ListTransactionsForMonth(ctx,
		txnFilters{}.exportParams(userID, pgval.Date(first), pgval.Date(last.AddDate(0, 0, 1))))
	if err != nil {
		return 0, err
	}
	seen := make(map[string]int, len(existing))
	for _, t := range existing {
		seen[duplicateKey(t.OccurredOn.Time, t.Type, t.CategoryID, t.Amount, t.Description)]++
	}

	count := 0
	for _, row := range plan.Rows {
		if row.CategoryID == 0 {
			continue
		}
		key := duplicateKey(row.Date, row.Type, row.CategoryID, row.Amount, row.Note)
		if seen[key] > 0 {
			seen[key]--
			count++
		}
	}
	return count, nil
}

func duplicateKey(date time.Time, txnType string, categoryID, amount int64, note string) string {
	return date.Format("2006-01-02") + "\x00" + txnType + "\x00" +
		strconv.FormatInt(categoryID, 10) + "\x00" + strconv.FormatInt(amount, 10) + "\x00" + note
}

// applyImport mirrors handlers.applyImport exactly.
func applyImport(ctx context.Context, deps Deps, userID int64, plan *csvimport.Import) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := deps.Queries.WithTx(tx)

	created := make(map[string]int64, len(plan.NewCategories))
	for i, c := range plan.NewCategories {
		category, err := qtx.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
			UserID: pgval.Int64(userID), Name: c.Name, Type: c.Type,
			Color: categorySwatches[i%len(categorySwatches)],
		})
		if err != nil {
			return fmt.Errorf("create category %q: %w", c.Name, err)
		}
		created[csvimport.MatchKey(c.Name, c.Type)] = category.ID
	}

	for _, row := range plan.Rows {
		categoryID := row.CategoryID
		if categoryID == 0 {
			categoryID = created[csvimport.MatchKey(row.CategoryName, row.Type)]
		}
		if _, err := qtx.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
			UserID: userID, CategoryID: categoryID, Amount: row.Amount, Type: row.Type,
			Description: row.Note, OccurredOn: pgval.Date(row.Date),
		}); err != nil {
			return fmt.Errorf("create transaction from line %d: %w", row.Line, err)
		}
	}
	return tx.Commit(ctx)
}

func latestMonth(plan *csvimport.Import) string {
	latest := plan.Rows[0].Date
	for _, row := range plan.Rows {
		if row.Date.After(latest) {
			latest = row.Date
		}
	}
	return latest.Format("2006-01")
}

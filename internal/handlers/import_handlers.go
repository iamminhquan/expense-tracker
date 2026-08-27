package handlers

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

	"expensetracker/internal/auth"
	"expensetracker/internal/csvimport"
	"expensetracker/internal/sqlcgen"
)

// importMaxBytes caps an upload. A file at this size is already far past
// csvimport.MaxRows, so the limit that actually bites is the row count --
// this one is only here so a multi-gigabyte body is refused before it is
// read into memory rather than after.
const importMaxBytes = 1 << 20

// importPage renders the upload form. It does not render
// mobile_page_header: that block is driven by ActiveNav, and for
// "transactions" it would put a month picker and an add button on a page
// that has neither a month nor a list.
func importPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, r, deps, "import", "transactions", map[string]any{})
	}
}

// importHandler answers both steps of the import. The first post describes
// what the file would do; the second, carrying confirm and the fingerprint
// the first one issued, does it.
//
// The file is uploaded again for the second step rather than held anywhere
// between them: the form stays in the DOM with the file still selected, so
// the browser can re-send it, and that leaves no half-finished import on the
// server for a session that walked away. The fingerprint is what makes the
// re-upload safe -- without it, swapping the file after reading the preview
// would import something nobody looked at.
func importHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		r.Body = http.MaxBytesReader(w, r.Body, importMaxBytes)

		file, _, err := r.FormFile("file")
		if err != nil {
			importFailed(w, r, deps, "That file could not be read. Choose a .csv file smaller than 1 MB.")
			return
		}
		defer file.Close()

		// The file is always sniffed, even when a mapping was submitted: it
		// is what column indexes are validated against, and what the mapping
		// screen is re-rendered from if they do not hold up.
		sheet, err := csvimport.Sniff(file)
		if err != nil {
			importFailed(w, r, deps, planFailureMessage(err))
			return
		}

		catalog, err := importCatalog(r.Context(), deps, userID)
		if err != nil {
			log.Printf("import: load categories: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		mapping, submitted := mappingFromForm(r)
		switch {
		case r.FormValue("back") != "":
			// Coming back from the preview keeps the mapping that was tried,
			// so a wrong date order is one select away from fixed rather
			// than a fresh start.
			if !submitted {
				mapping = sheet.Guess
			}
			importMapping(w, r, deps, sheet, mapping, "")
			return
		case !submitted && sheet.Exact:
			// A file this app wrote needs no mapping screen.
			mapping = sheet.Guess
		case !submitted:
			importMapping(w, r, deps, sheet, sheet.Guess, "")
			return
		default:
			if msg := validateMapping(mapping, len(sheet.Columns)); msg != "" {
				importMapping(w, r, deps, sheet, mapping, msg)
				return
			}
		}

		// Sniff read the file to the end; Plan needs the same bytes again.
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			log.Printf("import: rewind upload: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		plan, err := csvimport.Plan(file, mapping, catalog, time.Now().In(vietnamLocation))
		if err != nil {
			importFailed(w, r, deps, planFailureMessage(err))
			return
		}

		duplicates, err := countImportDuplicates(r.Context(), deps, userID, plan)
		if err != nil {
			log.Printf("import: count duplicates: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if r.FormValue("confirm") == "" {
			renderNamed(w, r, deps, "import", "import_preview", "", previewData(plan, mapping, duplicates))
			return
		}

		// Both checks re-run against this upload rather than trusting what
		// the preview decided: the preview's answer travelled through the
		// browser, and only the file in hand can be imported.
		if r.FormValue("fingerprint") != plan.Fingerprint {
			importFailed(w, r, deps, "This is not the file you previewed. Preview it again before importing.")
			return
		}
		if !importable(plan) {
			importFailed(w, r, deps, "This file still has lines that cannot be imported.")
			return
		}

		if err := applyImport(r.Context(), deps, userID, plan); err != nil {
			log.Printf("import: apply: %v", err)
			importFailed(w, r, deps, "The import could not be saved. Nothing was changed, please try again.")
			return
		}

		// The list is sent to the month the newest imported row belongs to,
		// because a file of last March's transactions is otherwise imported
		// into a month the user is not looking at.
		month := latestMonth(plan)
		w.Header().Set("HX-Redirect", fmt.Sprintf("/transactions?month=%s&imported=%d", month, len(plan.Rows)))
	}
}

// maxShownErrors caps the error list. A file that is wrong in the same way
// on every line would otherwise answer a preview with thousands of
// identical complaints, and the first twenty already say what to fix.
const maxShownErrors = 20

// previewData is what import_preview renders. The plan is unpacked here
// rather than handed over whole, so the template neither reaches into
// another package's struct nor has to do the arithmetic of trimming a long
// error list.
func previewData(plan *csvimport.Import, m csvimport.Mapping, duplicates int) map[string]any {
	shown, more := plan.Errors, 0
	if len(shown) > maxShownErrors {
		shown, more = shown[:maxShownErrors], len(plan.Errors)-maxShownErrors
	}
	return map[string]any{
		"RowCount":      len(plan.Rows),
		"NewCategories": plan.NewCategories,
		"Errors":        shown,
		"MoreErrors":    more,
		"Rounded":       plan.Rounded,
		"Duplicates":    duplicates,
		"Fingerprint":   plan.Fingerprint,
		"Importable":    importable(plan),
		"DateSuspect":   mostlyDateErrors(plan.Errors),
		"Mapping":       mappingFieldValues(m),
	}
}

// mostlyDateErrors reports whether the failures look like one wrong answer
// on the mapping screen rather than a file full of bad lines. Picking the
// wrong date order is the easiest mistake to make there and the hardest to
// see afterwards, since every line then fails for what reads like its own
// reason.
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

// importable reports whether a plan can be applied at all: one bad line
// blocks the whole file, so that fixing it and re-importing cannot double
// the lines that were already fine.
func importable(plan *csvimport.Import) bool {
	return len(plan.Errors) == 0 && len(plan.Rows) > 0
}

func planFailureMessage(err error) string {
	if errors.Is(err, csvimport.ErrTooManyRows) {
		return fmt.Sprintf("This file has more than %d rows. Split it and import the parts.", csvimport.MaxRows)
	}
	return "This file is not shaped like the CSV $pend exports: " + err.Error() + "."
}

func importFailed(w http.ResponseWriter, r *http.Request, deps Deps, msg string) {
	renderNamed(w, r, deps, "import", "import_preview", "", map[string]any{"FatalError": msg})
}

// importCatalog is the account's categories as csvimport sees them: ids,
// names, slugs and types, and nothing about pgtype.
func importCatalog(ctx context.Context, deps Deps, userID int64) ([]csvimport.Category, error) {
	rows, err := deps.Queries.ListCategoriesForUser(ctx, pgInt64(userID))
	if err != nil {
		return nil, err
	}
	catalog := make([]csvimport.Category, 0, len(rows))
	for _, c := range rows {
		catalog = append(catalog, csvimport.Category{
			ID: c.ID, Name: c.Name, Slug: c.Slug.String, Type: c.Type,
		})
	}
	return catalog, nil
}

// countImportDuplicates reports how many planned rows already exist exactly
// as they stand. It is a warning, not a rule: two identical coffees on one
// day are a real thing to record, so the count is shown and the import goes
// ahead if the user still wants it.
//
// The existing rows are read once over the file's own date range rather than
// row by row, and each match is spent so that two identical lines in the
// file need two matching rows on file to both count as duplicates.
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
		txnFilters{}.exportParams(userID, pgDate(first), pgDate(last.AddDate(0, 0, 1))))
	if err != nil {
		return 0, err
	}
	seen := make(map[string]int, len(existing))
	for _, t := range existing {
		seen[duplicateKey(t.OccurredOn.Time, t.Type, t.CategoryID, t.Amount, t.Description)]++
	}

	count := 0
	for _, row := range plan.Rows {
		// A category that does not exist yet can have no history behind it.
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

// applyImport writes the whole plan or none of it. The new categories go in
// first because the rows that named them have nothing to point at until
// they exist.
func applyImport(ctx context.Context, deps Deps, userID int64, plan *csvimport.Import) error {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := deps.Queries.WithTx(tx)

	created := make(map[string]int64, len(plan.NewCategories))
	for i, c := range plan.NewCategories {
		// The palette is walked in order rather than chosen: the colors are
		// interchangeable, and a file's categories at least come out
		// distinguishable from each other.
		category, err := qtx.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
			UserID: pgInt64(userID), Name: c.Name, Type: c.Type,
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
			Description: row.Note, OccurredOn: pgDate(row.Date),
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

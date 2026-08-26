package handlers

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"strconv"

	"expensetracker/internal/auth"
	"expensetracker/internal/i18n"
)

// exportColumns is the CSV header. Date is ISO rather than dateShort's
// "11 Aug" and Amount is a bare integer rather than vnd's "45,000₫",
// because this file is read by a spreadsheet, not a person: a formatted
// amount lands in a cell as text and will not sum. The sign lives in Type,
// which keeps Amount usable in a pivot table without a second parse.
var exportColumns = []string{"Date", "Type", "Category", "Amount", "Note"}

// utf8BOM is written ahead of the header because Excel assumes the host
// codepage for a CSV that lacks one, which turns every Vietnamese note and
// the ₫ in a category name into mojibake. Everything else that reads CSV
// tolerates the mark, so it is cheaper than the bug report.
const utf8BOM = "\ufeff"

// exportTransactionsHandler serves the transactions the list is currently
// showing as a CSV download: the same month, narrowed by the same filters.
//
// The filters come from the request's own URL rather than HX-Current-URL --
// unlike a mutation, this is a real top-level navigation the browser makes
// with the query string already on it, since htmx cannot hand an XHR
// response to the download manager.
func exportTransactionsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		scope := newTxnScope(r.URL.Query().Get("month"))
		from, to := scope.Bounds()
		filters := filtersFromRequest(r)

		// Read everything before writing a byte: once the first row is on the
		// wire the status line is already sent, and a mid-stream failure can
		// no longer be reported as a 500.
		rows, err := deps.Queries.ListTransactionsForMonth(r.Context(), filters.exportParams(userID, from, to))
		if err != nil {
			log.Printf("export transactions: %v", err)
			http.Error(w, "could not export transactions", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		// The filename names the scope rather than a month, so an all-months
		// download does not land in the folder claiming to be one August.
		w.Header().Set("Content-Disposition", `attachment; filename="spend-`+scope.Value+`.csv"`)

		if _, err := io.WriteString(w, utf8BOM); err != nil {
			log.Printf("export transactions: write bom: %v", err)
			return
		}

		out := csv.NewWriter(w)
		records := make([][]string, 0, len(rows)+1)
		records = append(records, exportColumns)
		for _, row := range rows {
			records = append(records, []string{
				row.OccurredOn.Time.Format("2006-01-02"),
				row.Type,
				i18n.CategoryName(row.CategorySlug, row.CategoryName),
				strconv.FormatInt(row.Amount, 10),
				row.Description,
			})
		}
		if err := out.WriteAll(records); err != nil {
			// The response is already committed, so this can only be logged.
			log.Printf("export transactions: write csv: %v", err)
		}
	}
}

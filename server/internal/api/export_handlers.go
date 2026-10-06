package api

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"strconv"

	"expensetracker/internal/i18n"

	"github.com/gin-gonic/gin"
)

// exportColumns and utf8BOM mirror handlers/txn_export.go exactly -- see
// its comments for why Amount is a bare integer (a spreadsheet has to sum
// it, not read it) and why the BOM is written at all (Excel mojibakes
// Vietnamese text without one).
var exportColumns = []string{"Date", "Type", "Category", "Amount", "Note"}

const utf8BOM = "\ufeff"

// exportTransactionsHandler mirrors handlers.exportTransactionsHandler
// exactly: the same month, narrowed by the same filters, as a CSV
// download. Unlike the HTML side, a plain <a href> can't carry this
// request's Authorization header -- the client fetches it with its access
// token attached and turns the response into a download itself (an
// object URL + a synthetic <a download>), the standard SPA pattern for an
// authenticated file download.
func exportTransactionsHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		query := c.Request.URL.Query()
		scope := newMonthScope(query.Get("month"))
		from, to := scope.Bounds()
		filters := filtersFromQuery(query)

		rows, err := deps.Queries.ListTransactionsForMonth(c.Request.Context(), filters.exportParams(userID, from, to))
		if err != nil {
			log.Printf("export transactions: %v", err)
			respondError(c, http.StatusInternalServerError, "could not export transactions")
			return
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="spend-`+scope.Value+`.csv"`)
		c.Status(http.StatusOK)

		if _, err := io.WriteString(c.Writer, utf8BOM); err != nil {
			log.Printf("export transactions: write bom: %v", err)
			return
		}
		out := csv.NewWriter(c.Writer)
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
			log.Printf("export transactions: write csv: %v", err)
		}
	}
}

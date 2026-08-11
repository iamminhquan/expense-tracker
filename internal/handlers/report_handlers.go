package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)

		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
			UserID:       userID,
			OccurredOn:   pgDate(from),
			OccurredOn_2: pgDate(to),
		})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}

		breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{
			UserID:       userID,
			OccurredOn:   pgDate(from),
			OccurredOn_2: pgDate(to),
		})
		if err != nil {
			http.Error(w, "could not load breakdown", http.StatusInternalServerError)
			return
		}

		labels := make([]string, len(breakdown))
		values := make([]int64, len(breakdown))
		colors := make([]string, len(breakdown))
		for i, row := range breakdown {
			labels[i] = row.CategoryName
			values[i] = row.Total
			colors[i] = row.CategoryColor
		}
		labelsJSON, _ := json.Marshal(labels)
		valuesJSON, _ := json.Marshal(values)
		colorsJSON, _ := json.Marshal(colors)

		deps.Templates["dashboard"].ExecuteTemplate(w, "layout", map[string]any{
			"TotalExpense":        totals.TotalExpense,
			"TotalIncome":         totals.TotalIncome,
			"BreakdownLabelsJSON": template.JS(labelsJSON),
			"BreakdownValuesJSON": template.JS(valuesJSON),
			"BreakdownColorsJSON": template.JS(colorsJSON),
		})
	}
}

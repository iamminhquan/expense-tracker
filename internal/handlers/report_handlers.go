package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		from, to := currentMonthRange()

		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
			UserID:       userID,
			OccurredOn:   from,
			OccurredOn_2: to,
		})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}

		breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{
			UserID:       userID,
			OccurredOn:   from,
			OccurredOn_2: to,
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

		render(w, deps, "dashboard", map[string]any{
			"TotalExpense":        totals.TotalExpense,
			"TotalIncome":         totals.TotalIncome,
			"BreakdownLabelsJSON": template.JS(labelsJSON),
			"BreakdownValuesJSON": template.JS(valuesJSON),
			"BreakdownColorsJSON": template.JS(colorsJSON),
		})
	}
}

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgDate converts a parsed calendar date into the pgtype.Date that sqlc
// generates for a DATE column.
func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		var formErr string
		if r.Method == http.MethodPost {
			categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
			if err != nil {
				http.Error(w, "invalid category", http.StatusBadRequest)
				return
			}
			amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
			if err != nil || amount <= 0 {
				http.Error(w, "invalid amount", http.StatusBadRequest)
				return
			}
			occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
			if err != nil {
				http.Error(w, "invalid date", http.StatusBadRequest)
				return
			}
			txnType := r.FormValue("type")
			if txnType != "expense" && txnType != "income" {
				http.Error(w, "invalid type", http.StatusBadRequest)
				return
			}

			_, err = deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
				UserID:      userID,
				CategoryID:  categoryID,
				Amount:      amount,
				Type:        txnType,
				Description: r.FormValue("description"),
				OccurredOn:  pgDate(occurredOn),
			})
			if err != nil {
				formErr = "could not create transaction"
			}
		}

		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)

		transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
			UserID:       userID,
			OccurredOn:   pgDate(from),
			OccurredOn_2: pgDate(to),
		})
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		deps.Templates["transactions"].ExecuteTemplate(w, "layout", map[string]any{
			"Transactions": transactions,
			"Categories":   categories,
			"Error":        formErr,
		})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{
			ID:     id,
			UserID: userID,
		}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}

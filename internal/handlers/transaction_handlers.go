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

			// Verify the client-supplied category_id is either owned by the
			// authenticated user or a shared default category (user_id IS
			// NULL) before attaching it to a transaction. Without this check
			// a forged request could attach a transaction to another user's
			// private category, leaking its name/color via the joined
			// ListTransactionsForMonth query.
			if _, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
				ID:     categoryID,
				UserID: pgInt64(userID),
			}); err != nil {
				http.Error(w, "category not found", http.StatusForbidden)
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

		from, to := currentMonthRange()

		transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
			UserID:       userID,
			OccurredOn:   from,
			OccurredOn_2: to,
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

		render(w, deps, "transactions", map[string]any{
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

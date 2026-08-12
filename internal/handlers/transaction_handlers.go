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

			category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
				ID:     categoryID,
				UserID: pgInt64(userID),
			})
			if err != nil {
				http.Error(w, "category not found", http.StatusForbidden)
				return
			}

			switch {
			case category.Type != txnType:
				formErr = "Loại giao dịch không khớp với danh mục đã chọn"
			case len([]rune(r.FormValue("description"))) > 200:
				formErr = "Ghi chú tối đa 200 ký tự"
			case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
				formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
			default:
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

		render(w, r, deps, "transactions", "transactions", map[string]any{
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

package handlers

import (
	"fmt"
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

func monthLabel(t time.Time) string      { return fmt.Sprintf("Tháng %d, %d", int(t.Month()), t.Year()) }
func monthLabelLower(t time.Time) string { return fmt.Sprintf("tháng %d", int(t.Month())) }

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateTransaction(w, r, deps, userID)
			return
		}

		renderTransactionsPage(w, r, deps, userID, "", "")
	}
}

func renderTransactionsPage(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, quickAddError, selectedType string) {
	from, to := currentMonthRange()

	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}

	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	typ := selectedType
	if typ != "income" {
		typ = "expense"
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == typ {
			filteredCategories = append(filteredCategories, c)
		}
	}

	render(w, r, deps, "transactions", "transactions", map[string]any{
		"Transactions":    transactions,
		"Categories":      filteredCategories,
		"SelectedType":    selectedType,
		"TotalExpense":    totals.TotalExpense,
		"TotalIncome":     totals.TotalIncome,
		"Remaining":       totals.TotalIncome - totals.TotalExpense,
		"MonthLabel":      monthLabel(from.Time),
		"MonthLabelLower": monthLabelLower(from.Time),
		"Today":           time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError":   quickAddError,
	})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
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
		ID: categoryID, UserID: pgInt64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	var formErr string
	switch {
	case category.Type != txnType:
		formErr = "Loại giao dịch không khớp với danh mục đã chọn"
	case len([]rune(r.FormValue("description"))) > 200:
		formErr = "Ghi chú tối đa 200 ký tự"
	case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
		formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
	}
	if formErr != "" {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, formErr, txnType)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: categoryID, Amount: amount, Type: txnType,
		Description: r.FormValue("description"), OccurredOn: pgDate(occurredOn),
	})
	if err != nil {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, "Không thể thêm giao dịch, vui lòng thử lại.", txnType)
		return
	}

	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}
	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
		"Row": map[string]any{
			"ID": created.ID, "CategoryName": category.Name, "CategoryColor": category.Color,
			"Description": created.Description, "OccurredOn": created.OccurredOn,
			"Amount": created.Amount, "Type": created.Type,
		},
		"Totals": map[string]any{
			"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
			"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
		},
	})
}

// renderTransactionsPageForm re-renders just the quick_add_form fragment
// (targeted via HX-Retarget by the caller) after a validation failure,
// with the category list filtered to match the type the user had selected
// so their in-progress selection stays coherent.
func renderTransactionsPageForm(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, errMsg, selectedType string) {
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == selectedType {
			filteredCategories = append(filteredCategories, c)
		}
	}
	renderNamed(w, r, deps, "transactions", "quick_add_form", "", map[string]any{
		"Categories":    filteredCategories,
		"SelectedType":  selectedType,
		"Today":         time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError": errMsg,
	})
}

func categoryOptionsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		typ := r.FormValue("type")
		if typ != "income" {
			typ = "expense"
		}
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		var filtered []sqlcgen.Category
		for _, c := range categories {
			if c.Type == typ {
				filtered = append(filtered, c)
			}
		}
		renderNamed(w, r, deps, "transactions", "category_options", "", map[string]any{"Categories": filtered})
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

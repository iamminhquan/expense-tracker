package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// categorySwatches is the fixed 8-color palette SPEC.md section 1 offers in
// the category color picker. #A1A1AA (the 9th seeded color) is deliberately
// excluded here -- it's reserved for the "Other" default and the chart's
// synthetic "Other" aggregation, never user-selectable.
var categorySwatches = []string{
	"#D97757", "#5B8DEF", "#8B7BD8", "#6BA292",
	"#E0A82E", "#D97AA0", "#4FA871", "#7CA65C",
}

func isValidSwatch(color string) bool {
	for _, c := range categorySwatches {
		if c == color {
			return true
		}
	}
	return false
}

// categoryRowData builds the flat map category_row.html and
// category_row_edit.html both expect. Every response path that renders a
// row goes through this so the template never has to distinguish a raw
// sqlc struct from a hand-built one -- see Task 4's design notes for why
// that distinction matters for the optional OOBTarget field.
// slug is what tells a shared default apart from a category the user made:
// the template resolves the display name through catName, which needs it,
// and a map key that is simply absent makes html/template fail the whole
// render rather than fall back -- so every path must pass it, NULL included.
func categoryRowData(id int64, userID pgtype.Int8, slug pgtype.Text, name, typ, color string, txnCount int64, oobTarget string) map[string]any {
	return map[string]any{
		"ID": id, "UserID": userID, "Slug": slug, "Name": name, "Type": typ, "Color": color,
		"TransactionCount": txnCount, "OOBTarget": oobTarget,
	}
}

func categoriesPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateCategory(w, r, deps, userID)
			return
		}

		rows, err := deps.Queries.ListCategoriesWithTransactionCounts(r.Context(), userID)
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		var expense, income []map[string]any
		hasCustom := false
		for _, row := range rows {
			data := categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount, "")
			if row.Type == "expense" {
				expense = append(expense, data)
			} else {
				income = append(income, data)
			}
			if row.UserID.Valid {
				hasCustom = true
			}
		}

		render(w, r, deps, "categories", "categories", map[string]any{
			"ExpenseCategories":   expense,
			"IncomeCategories":    income,
			"HasCustomCategories": hasCustom,
		})
	}
}

func handleCreateCategory(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	name := strings.TrimSpace(r.FormValue("name"))
	typ := r.FormValue("type")
	color := r.FormValue("color")

	fail := func(msg string) {
		w.Header().Set("HX-Retarget", "#add-category-form")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderNamed(w, r, deps, "categories", "add_category_form", "", map[string]any{
			"CategoryError": msg, "CategoryName": name, "CategoryType": typ,
		})
	}

	if name == "" {
		fail("Vui lòng nhập tên danh mục.")
		return
	}
	if typ != "expense" && typ != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	if !isValidSwatch(color) {
		http.Error(w, "invalid color", http.StatusBadRequest)
		return
	}

	created, err := deps.Queries.CreateCategory(r.Context(), sqlcgen.CreateCategoryParams{
		UserID: pgInt64(userID),
		Name:   name,
		Type:   typ,
		Color:  color,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail("Đã có danh mục tên này.")
			return
		}
		log.Printf("create category: %v", err)
		fail("Không thể tạo danh mục, vui lòng thử lại.")
		return
	}

	renderNamed(w, r, deps, "categories", "category_create_response", "", map[string]any{
		"Row": categoryRowData(
			created.ID, created.UserID, created.Slug, created.Name, created.Type, created.Color, 0, "#category-list-"+created.Type,
		),
		// Creating a category always means the user now has at least one
		// custom category, so the "no custom categories yet" empty state
		// (see categories.html/#categories-empty) must be hidden.
		"HasCustomCategories": true,
	})
}

func updateCategoryColorHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		color := r.FormValue("color")
		if !isValidSwatch(color) {
			http.Error(w, "invalid color", http.StatusBadRequest)
			return
		}

		// UpdateCategoryColor's WHERE clause matches a row owned by this
		// user OR a shared default (user_id IS NULL) -- SPEC.md section 4.1
		// explicitly lets defaults have a working "Đổi màu" action with no
		// ownership carve-out, unlike rename/delete.
		updated, err := deps.Queries.UpdateCategoryColor(r.Context(), sqlcgen.UpdateCategoryColorParams{
			ID: id, UserID: pgInt64(userID), Color: color,
		})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: updated.ID, UserID: userID})
		if err != nil {
			log.Printf("update category color: count transactions: %v", err)
		}
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(updated.ID, updated.UserID, updated.Slug, updated.Name, updated.Type, updated.Color, count, ""))
	}
}

func editCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := deps.Queries.GetCategoryWithTransactionCount(r.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !row.UserID.Valid {
			http.Error(w, "default categories cannot be renamed", http.StatusForbidden)
			return
		}
		renderNamed(w, r, deps, "categories", "category_row_edit", "", categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount, ""))
	}
}

func viewCategoryRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := deps.Queries.GetCategoryWithTransactionCount(r.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount, ""))
	}
}

func updateCategoryNameHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		existing, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !existing.UserID.Valid {
			http.Error(w, "default categories cannot be renamed", http.StatusForbidden)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			data := categoryRowData(existing.ID, existing.UserID, existing.Slug, existing.Name, existing.Type, existing.Color, 0, "")
			data["Error"] = "Vui lòng nhập tên danh mục."
			renderNamed(w, r, deps, "categories", "category_row_edit", "", data)
			return
		}

		updated, err := deps.Queries.UpdateCategoryName(r.Context(), sqlcgen.UpdateCategoryNameParams{ID: id, UserID: pgInt64(userID), Name: name})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				data := categoryRowData(existing.ID, existing.UserID, existing.Slug, name, existing.Type, existing.Color, 0, "")
				data["Error"] = "Đã có danh mục tên này."
				renderNamed(w, r, deps, "categories", "category_row_edit", "", data)
				return
			}
			log.Printf("update category name: %v", err)
			http.Error(w, "could not rename category", http.StatusInternalServerError)
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: updated.ID, UserID: userID})
		if err != nil {
			log.Printf("update category name: count transactions: %v", err)
		}
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(updated.ID, updated.UserID, updated.Slug, updated.Name, updated.Type, updated.Color, count, ""))
	}
}

// respondCategoryDeleted renders the categories_empty_oob fragment so a
// delete that empties the user's last custom category brings the "you have
// no custom categories yet" empty state (categories.html/#categories-empty)
// back, and a delete that still leaves other custom categories keeps it
// hidden. Its response body is entirely OOB content with nothing for the
// hx-target="#category-row-{id}" outerHTML swap the delete button issued,
// so that row is simply removed from the DOM -- mirroring the previous
// empty-body-means-"remove the row" behavior.
func respondCategoryDeleted(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	all, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		log.Printf("delete category: list categories for empty-state check: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	hasCustom := false
	for _, c := range all {
		if c.UserID.Valid {
			hasCustom = true
			break
		}
	}
	renderNamed(w, r, deps, "categories", "categories_empty_oob", "", map[string]any{"HasCustomCategories": hasCustom})
}

func deleteCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !category.UserID.Valid {
			http.Error(w, "default categories cannot be deleted", http.StatusForbidden)
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: id, UserID: userID})
		if err != nil {
			http.Error(w, "could not check category usage", http.StatusInternalServerError)
			return
		}

		if count > 0 && category.Type == "income" {
			// No income-side "Other" exists in the 9-category default set to
			// reassign into (confirmed with the human via AskUserQuestion),
			// so an income category with existing transactions can't be
			// deleted at all.
			http.Error(w, "category has existing transactions", http.StatusConflict)
			return
		}

		if count > 0 {
			tx, err := deps.DB.Begin(r.Context())
			if err != nil {
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(r.Context())
			qtx := deps.Queries.WithTx(tx)

			khac, err := qtx.GetDefaultCategoryForReassignment(r.Context())
			if err != nil {
				log.Printf("delete category: load Other default: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.ReassignCategoryTransactions(r.Context(), sqlcgen.ReassignCategoryTransactionsParams{
				CategoryID: khac.ID, CategoryID_2: id, UserID: userID,
			}); err != nil {
				log.Printf("delete category: reassign transactions: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgInt64(userID)}); err != nil {
				log.Printf("delete category: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				log.Printf("delete category: commit: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			respondCategoryDeleted(w, r, deps, userID)
			return
		}

		if _, err := deps.Queries.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgInt64(userID)}); err != nil {
			log.Printf("delete category: %v", err)
			http.Error(w, "could not delete category", http.StatusInternalServerError)
			return
		}
		respondCategoryDeleted(w, r, deps, userID)
	}
}

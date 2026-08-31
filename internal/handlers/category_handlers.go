package handlers

import (
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// categorySwatches is the fixed 8-color palette the category color picker
// offers. #A1A1AA (the 9th seeded color) is deliberately excluded here --
// it's reserved for the "Other" default and the chart's synthetic "Other"
// aggregation, never user-selectable.
var categorySwatches = []string{
	"#D97757", "#5B8DEF", "#8B7BD8", "#6BA292",
	"#E0A82E", "#D97AA0", "#4FA871", "#7CA65C",
}

func isValidSwatch(color string) bool {
	return slices.Contains(categorySwatches, color)
}

// categoryRow is one category as category_row.html and
// category_row_edit.html both see it. Every response path that renders a
// row builds one, so the template never has to distinguish a raw sqlc
// struct from a hand-built one.
//
// The out-of-band target of a newly created row is deliberately not a field
// here: it belongs to the wrapper category_create_response puts *around*
// the row, never to the row itself -- see the comment there.
//
// Slug is what tells a shared default apart from a category the user made,
// and the template resolves the display name through catName, which needs
// it. It was the argument for making this a struct: as a map, a response
// path that left the key out rendered a nameless row and reported nothing,
// because html/template prints nothing at all for a key its map lacks.
type categoryRow struct {
	ID               int64
	UserID           pgtype.Int8
	Slug             pgtype.Text
	Name             string
	Type             string
	Color            string
	TransactionCount int64
	// Error is the message a rejected rename shows beside the input. Only
	// category_row_edit reads it; the read-only row leaves it empty.
	Error string
}

// categoryRowData builds the row every category response renders through.
func categoryRowData(id int64, userID pgtype.Int8, slug pgtype.Text, name, typ, color string, txnCount int64) categoryRow {
	return categoryRow{
		ID: id, UserID: userID, Slug: slug, Name: name, Type: typ, Color: color,
		TransactionCount: txnCount,
	}
}

// categoryCreateResponse is what a successful create answers with: the new
// row, the list it is prepended into, and the empty state that has to
// disappear now that the account has a category of its own.
type categoryCreateResponse struct {
	Row                 categoryRow
	OOBTarget           string
	HasCustomCategories bool
}

// categoriesEmptyState is the out-of-band swap that brings the "you have no
// custom categories yet" panel back, or keeps it hidden.
type categoriesEmptyState struct {
	HasCustomCategories bool
}

// categoriesView is the categories page: the two lists, and whether the
// account has a category of its own yet -- which is what decides between
// the list and the "you haven't created any categories" panel.
type categoriesView struct {
	viewData
	// The page renders the create form inline, so it has to carry the
	// form's own fields too -- embedded rather than restated, which is what
	// keeps the two renders of that form asking for the same set. The page
	// leaves them zero; only a rejected submit fills any of them in, and it
	// re-renders the form on its own through addCategoryForm.
	addCategoryForm

	ExpenseCategories   []categoryRow
	IncomeCategories    []categoryRow
	HasCustomCategories bool
}

// addCategoryForm is the create form's own state, re-rendered in place when
// a submission is rejected.
type addCategoryForm struct {
	CategoryError string
	CategoryName  string
	CategoryType  string
}

// categoriesPage returns an HTTP handler that creates categories for POST requests and renders expense and income categories for other requests.
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

		var expense, income []categoryRow
		hasCustom := false
		for _, row := range rows {
			data := categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount)
			if row.Type == "expense" {
				expense = append(expense, data)
			} else {
				income = append(income, data)
			}
			if row.UserID.Valid {
				hasCustom = true
			}
		}

		render(w, r, deps, "categories", "categories", &categoriesView{
			ExpenseCategories:   expense,
			IncomeCategories:    income,
			HasCustomCategories: hasCustom,
		})
	}
}

// handleCreateCategory validates and creates a category for the user, rendering form
// errors or the newly created category row as appropriate.
func handleCreateCategory(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	name := strings.TrimSpace(r.FormValue("name"))
	typ := r.FormValue("type")
	color := r.FormValue("color")

	fail := func(msg string) {
		w.Header().Set("HX-Retarget", "#add-category-form")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderFragment(w, r, deps, "categories", "add_category_form", addCategoryForm{
			CategoryError: msg, CategoryName: name, CategoryType: typ,
		})
	}

	if name == "" {
		fail("Please enter a category name.")
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
		UserID: pgval.Int64(userID),
		Name:   name,
		Type:   typ,
		Color:  color,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail("You already have a category with that name.")
			return
		}
		log.Printf("create category: %v", err)
		fail("Could not create the category, please try again.")
		return
	}

	renderFragment(w, r, deps, "categories", "category_create_response", categoryCreateResponse{
		Row:       categoryRowData(created.ID, created.UserID, created.Slug, created.Name, created.Type, created.Color, 0),
		OOBTarget: "#category-list-" + created.Type,
		// Creating a category always means the user now has at least one
		// custom category, so the "no custom categories yet" empty state
		// (see categories.html/#categories-empty) must be hidden.
		HasCustomCategories: true,
	})
}

// updateCategoryColorHandler creates an HTTP handler that updates a category's color
// for its owner or a shared default category and renders the updated category row.
func updateCategoryColorHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}
		color := r.FormValue("color")
		if !isValidSwatch(color) {
			http.Error(w, "invalid color", http.StatusBadRequest)
			return
		}

		// UpdateCategoryColor's WHERE clause matches a row owned by this
		// user OR a shared default (user_id IS NULL): recolouring a default
		// is allowed for everyone, unlike rename/delete, which carve
		// defaults out.
		updated, err := deps.Queries.UpdateCategoryColor(r.Context(), sqlcgen.UpdateCategoryColorParams{
			ID: id, UserID: pgval.Int64(userID), Color: color,
		})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: updated.ID, UserID: userID})
		if err != nil {
			log.Printf("update category color: count transactions: %v", err)
		}
		renderFragment(w, r, deps, "categories", "category_row", categoryRowData(updated.ID, updated.UserID, updated.Slug, updated.Name, updated.Type, updated.Color, count))
	}
}

// editCategoryHandler renders the editable row for a user-owned category.
// It returns 404 when the category cannot be found and 403 when the category is a shared default.
func editCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
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
		renderFragment(w, r, deps, "categories", "category_row_edit", categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount))
	}
}

// viewCategoryRowHandler renders a user-visible category row or returns HTTP 404 when the category cannot be found.
func viewCategoryRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}
		row, err := deps.Queries.GetCategoryWithTransactionCount(r.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		renderFragment(w, r, deps, "categories", "category_row", categoryRowData(row.ID, row.UserID, row.Slug, row.Name, row.Type, row.Color, row.TransactionCount))
	}
}

// updateCategoryNameHandler creates an HTTP handler that validates and updates a user-owned category name, then renders the updated category row. Shared default categories cannot be renamed.
func updateCategoryNameHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		existing, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgval.Int64(userID)})
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
			row := categoryRowData(existing.ID, existing.UserID, existing.Slug, existing.Name, existing.Type, existing.Color, 0)
			row.Error = "Please enter a category name."
			renderFragment(w, r, deps, "categories", "category_row_edit", row)
			return
		}

		updated, err := deps.Queries.UpdateCategoryName(r.Context(), sqlcgen.UpdateCategoryNameParams{ID: id, UserID: pgval.Int64(userID), Name: name})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				row := categoryRowData(existing.ID, existing.UserID, existing.Slug, name, existing.Type, existing.Color, 0)
				row.Error = "You already have a category with that name."
				renderFragment(w, r, deps, "categories", "category_row_edit", row)
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
		renderFragment(w, r, deps, "categories", "category_row", categoryRowData(updated.ID, updated.UserID, updated.Slug, updated.Name, updated.Type, updated.Color, count))
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
	all, err := deps.Queries.ListCategoriesForUser(r.Context(), pgval.Int64(userID))
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
	renderFragment(w, r, deps, "categories", "categories_empty_oob", categoriesEmptyState{HasCustomCategories: hasCustom})
}

func deleteCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, ok := idParam(w, r)
		if !ok {
			return
		}

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgval.Int64(userID)})
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
			// An income-side "Other" default now exists (after migration 000014),
			// but GetDefaultCategoryForReassignment is hardcoded to the
			// expense-side `other` slug, so deleting an income category with
			// existing transactions is still refused rather than reassigned.
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

			other, err := qtx.GetDefaultCategoryForReassignment(r.Context())
			if err != nil {
				log.Printf("delete category: load Other default: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.ReassignCategoryTransactions(r.Context(), sqlcgen.ReassignCategoryTransactionsParams{
				CategoryID: other.ID, CategoryID_2: id, UserID: userID,
			}); err != nil {
				log.Printf("delete category: reassign transactions: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgval.Int64(userID)}); err != nil {
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

		if _, err := deps.Queries.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgval.Int64(userID)}); err != nil {
			log.Printf("delete category: %v", err)
			http.Error(w, "could not delete category", http.StatusInternalServerError)
			return
		}
		respondCategoryDeleted(w, r, deps, userID)
	}
}

package api

import (
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

// categorySwatches is handlers.categorySwatches duplicated verbatim -- the
// 8-color palette the picker offers. #A1A1AA (the 9th seeded color) stays
// reserved for the "Other" default and the dashboard chart's synthetic
// "Other" slice, never user-selectable, in both packages alike.
var categorySwatches = []string{
	"#D97757", "#5B8DEF", "#8B7BD8", "#6BA292",
	"#E0A82E", "#D97AA0", "#4FA871", "#7CA65C",
}

func isValidSwatch(color string) bool {
	return slices.Contains(categorySwatches, color)
}

// categoryDTO is one category as the client sees it. Name is already
// resolved through i18n.CategoryName -- the client has no copy of the
// slug->name table and shouldn't need one just to display a row.
// IsDefault (user_id IS NULL) is what the client checks before offering a
// rename/delete action: a default can be recolored by anyone but never
// renamed or deleted.
type categoryDTO struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Color            string `json:"color"`
	TransactionCount int64  `json:"transactionCount"`
	IsDefault        bool   `json:"isDefault"`
}

func newCategoryDTO(c sqlcgen.Category, txnCount int64) categoryDTO {
	return categoryDTO{
		ID:               c.ID,
		Name:             i18n.CategoryName(c.Slug, c.Name),
		Type:             c.Type,
		Color:            c.Color,
		TransactionCount: txnCount,
		IsDefault:        !c.UserID.Valid,
	}
}

// categoriesListResponse is GET /api/categories' body: both lists, plus
// whether the account has a category of its own -- the client-side
// equivalent of the HTML side's "you have no custom categories yet" empty
// state, computed the same way (a non-null user_id on any row).
type categoriesListResponse struct {
	ExpenseCategories   []categoryDTO `json:"expenseCategories"`
	IncomeCategories    []categoryDTO `json:"incomeCategories"`
	HasCustomCategories bool          `json:"hasCustomCategories"`
}

func listCategoriesHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		rows, err := deps.Queries.ListCategoriesWithTransactionCounts(c.Request.Context(), userID)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "could not load categories")
			return
		}

		// Both lists start empty (never nil) for the same reason
		// transaction_query.go's monthOptions and dashboard_handlers.go's
		// buildPieData do: encoding/json renders a nil slice as null, and
		// the client indexes straight into these arrays with no null
		// check. In practice the default categories mean neither list is
		// ever truly empty, but nothing guarantees that stays true.
		resp := categoriesListResponse{ExpenseCategories: []categoryDTO{}, IncomeCategories: []categoryDTO{}}
		for _, row := range rows {
			dto := categoryDTO{
				ID:               row.ID,
				Name:             i18n.CategoryName(row.Slug, row.Name),
				Type:             row.Type,
				Color:            row.Color,
				TransactionCount: row.TransactionCount,
				IsDefault:        !row.UserID.Valid,
			}
			if row.Type == "expense" {
				resp.ExpenseCategories = append(resp.ExpenseCategories, dto)
			} else {
				resp.IncomeCategories = append(resp.IncomeCategories, dto)
			}
			if row.UserID.Valid {
				resp.HasCustomCategories = true
			}
		}
		c.JSON(http.StatusOK, resp)
	}
}

type createCategoryRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

func createCategoryHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req createCategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, "malformed request body")
			return
		}
		req.Name = strings.TrimSpace(req.Name)

		if req.Name == "" {
			errorResponse(c, http.StatusBadRequest, "please enter a category name")
			return
		}
		if req.Type != "expense" && req.Type != "income" {
			errorResponse(c, http.StatusBadRequest, "type must be \"expense\" or \"income\"")
			return
		}
		if !isValidSwatch(req.Color) {
			errorResponse(c, http.StatusBadRequest, "invalid color")
			return
		}

		created, err := deps.Queries.CreateCategory(c.Request.Context(), sqlcgen.CreateCategoryParams{
			UserID: pgval.Int64(userID),
			Name:   req.Name,
			Type:   req.Type,
			Color:  req.Color,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				errorResponse(c, http.StatusConflict, "you already have a category with that name")
				return
			}
			log.Printf("create category: %v", err)
			errorResponse(c, http.StatusInternalServerError, "could not create the category, please try again")
			return
		}

		c.JSON(http.StatusCreated, newCategoryDTO(created, 0))
	}
}

// categoryIDParam parses the :id path param shared by every /categories/:id
// route below.
func categoryIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// updateCategoryRequest is PATCH /api/categories/:id's body. Both fields
// are optional and independent -- send Color alone to recolor (allowed on
// a default category too), Name alone to rename (never allowed on a
// default), or both. This is one endpoint where the HTML side has two
// (PATCH .../color and PATCH .../name) only because two different template
// fragments needed telling apart; a JSON response has no such seam.
type updateCategoryRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func updateCategoryHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		id, ok := categoryIDParam(c)
		if !ok {
			return
		}
		var req updateCategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errorResponse(c, http.StatusBadRequest, "malformed request body")
			return
		}
		if req.Name == nil && req.Color == nil {
			errorResponse(c, http.StatusBadRequest, "nothing to update")
			return
		}

		if req.Color != nil {
			if !isValidSwatch(*req.Color) {
				errorResponse(c, http.StatusBadRequest, "invalid color")
				return
			}
			// UpdateCategoryColor's WHERE clause matches a row owned by
			// this user OR a shared default (user_id IS NULL): recoloring
			// a default is allowed for everyone, unlike rename, which
			// carves defaults out below.
			if _, err := deps.Queries.UpdateCategoryColor(c.Request.Context(), sqlcgen.UpdateCategoryColorParams{
				ID: id, UserID: pgval.Int64(userID), Color: *req.Color,
			}); err != nil {
				errorResponse(c, http.StatusNotFound, "category not found")
				return
			}
		}

		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				errorResponse(c, http.StatusBadRequest, "please enter a category name")
				return
			}
			existing, err := deps.Queries.GetCategoryForUser(c.Request.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgval.Int64(userID)})
			if err != nil {
				errorResponse(c, http.StatusNotFound, "category not found")
				return
			}
			if !existing.UserID.Valid {
				errorResponse(c, http.StatusForbidden, "default categories cannot be renamed")
				return
			}
			if _, err := deps.Queries.UpdateCategoryName(c.Request.Context(), sqlcgen.UpdateCategoryNameParams{ID: id, UserID: pgval.Int64(userID), Name: name}); err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					errorResponse(c, http.StatusConflict, "you already have a category with that name")
					return
				}
				log.Printf("update category name: %v", err)
				errorResponse(c, http.StatusInternalServerError, "could not rename category")
				return
			}
		}

		row, err := deps.Queries.GetCategoryWithTransactionCount(c.Request.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: userID})
		if err != nil {
			errorResponse(c, http.StatusNotFound, "category not found")
			return
		}
		c.JSON(http.StatusOK, categoryDTO{
			ID: row.ID, Name: i18n.CategoryName(row.Slug, row.Name), Type: row.Type,
			Color: row.Color, TransactionCount: row.TransactionCount, IsDefault: !row.UserID.Valid,
		})
	}
}

// deleteCategoryHandler mirrors handlers.deleteCategoryHandler exactly:
// a default can't be deleted; an income category with existing
// transactions is refused (there's no income-side "Other" default to
// reassign to -- see GetDefaultCategoryForReassignment's doc comment);
// an expense category with existing transactions has them reassigned to
// the "Other" default in the same transaction as the delete.
func deleteCategoryHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		id, ok := categoryIDParam(c)
		if !ok {
			return
		}

		category, err := deps.Queries.GetCategoryForUser(c.Request.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgval.Int64(userID)})
		if err != nil {
			errorResponse(c, http.StatusNotFound, "category not found")
			return
		}
		if !category.UserID.Valid {
			errorResponse(c, http.StatusForbidden, "default categories cannot be deleted")
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(c.Request.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: id, UserID: userID})
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "could not check category usage")
			return
		}

		if count > 0 && category.Type == "income" {
			errorResponse(c, http.StatusConflict, "category has existing transactions")
			return
		}

		if count > 0 {
			tx, err := deps.DB.Begin(c.Request.Context())
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "could not delete category")
				return
			}
			defer tx.Rollback(c.Request.Context())
			qtx := deps.Queries.WithTx(tx)

			other, err := qtx.GetDefaultCategoryForReassignment(c.Request.Context())
			if err != nil {
				log.Printf("delete category: load Other default: %v", err)
				errorResponse(c, http.StatusInternalServerError, "could not delete category")
				return
			}
			if _, err := qtx.ReassignCategoryTransactions(c.Request.Context(), sqlcgen.ReassignCategoryTransactionsParams{
				CategoryID: other.ID, CategoryID_2: id, UserID: userID,
			}); err != nil {
				log.Printf("delete category: reassign transactions: %v", err)
				errorResponse(c, http.StatusInternalServerError, "could not delete category")
				return
			}
			if _, err := qtx.DeleteCategory(c.Request.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgval.Int64(userID)}); err != nil {
				log.Printf("delete category: %v", err)
				errorResponse(c, http.StatusInternalServerError, "could not delete category")
				return
			}
			if err := tx.Commit(c.Request.Context()); err != nil {
				log.Printf("delete category: commit: %v", err)
				errorResponse(c, http.StatusInternalServerError, "could not delete category")
				return
			}
			c.Status(http.StatusNoContent)
			return
		}

		if _, err := deps.Queries.DeleteCategory(c.Request.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgval.Int64(userID)}); err != nil {
			log.Printf("delete category: %v", err)
			errorResponse(c, http.StatusInternalServerError, "could not delete category")
			return
		}
		c.Status(http.StatusNoContent)
	}
}

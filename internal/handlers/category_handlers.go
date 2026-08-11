package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
)

func categoriesPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			_, err := deps.Queries.CreateCategory(r.Context(), sqlcgen.CreateCategoryParams{
				UserID: pgInt64(userID),
				Name:   r.FormValue("name"),
				Type:   r.FormValue("type"),
				Color:  r.FormValue("color"),
			})
			if err != nil {
				http.Error(w, "could not create category", http.StatusBadRequest)
				return
			}
		}

		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		render(w, deps, "categories", map[string]any{"Categories": categories})
	}
}

func deleteCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		idParam := chiURLParam(r, "id")
		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{
			ID:     id,
			UserID: pgInt64(userID),
		}); err != nil {
			// transactions.category_id has no ON DELETE clause (RESTRICT by
			// default), so deleting a category still referenced by existing
			// transactions raises a foreign-key violation (23503). Surface
			// that as a friendly message on the categories page instead of
			// a bare 500.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				categories, listErr := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
				if listErr != nil {
					http.Error(w, "could not load categories", http.StatusInternalServerError)
					return
				}
				render(w, deps, "categories", map[string]any{
					"Categories": categories,
					"Error":      "Không thể xóa danh mục đang được sử dụng bởi các giao dịch",
				})
				return
			}
			log.Printf("delete category: %v", err)
			http.Error(w, "could not delete category", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	}
}

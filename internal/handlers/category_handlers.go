package handlers

import (
	"net/http"
	"strconv"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
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

		deps.Templates["categories"].ExecuteTemplate(w, "layout", map[string]any{"Categories": categories})
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
			http.Error(w, "could not delete category", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	}
}

package handlers

import (
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/csrf"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(csrf.Middleware(deps.SecureCookies))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	r.Get("/register", registerPage(deps))
	r.Post("/register", registerPage(deps))
	r.Get("/login", loginPage(deps))
	r.Post("/login", loginPage(deps))
	r.Post("/logout", logoutHandler(deps))

	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireAuth(deps.Sessions, deps.CookieName))
		pr.Get("/categories", categoriesPage(deps))
		pr.Post("/categories", categoriesPage(deps))
		pr.Patch("/categories/{id}/color", updateCategoryColorHandler(deps))
		pr.Get("/categories/{id}/edit", editCategoryHandler(deps))
		pr.Get("/categories/{id}/view", viewCategoryRowHandler(deps))
		pr.Patch("/categories/{id}/name", updateCategoryNameHandler(deps))
		pr.Delete("/categories/{id}", deleteCategoryHandler(deps))
		pr.Get("/transactions", transactionsPage(deps))
		pr.Post("/transactions", transactionsPage(deps))
		pr.Get("/transactions/category-options", categoryOptionsHandler(deps))
		pr.Get("/transactions/{id}/edit", editTransactionRowHandler(deps))
		pr.Get("/transactions/{id}/view", viewTransactionRowHandler(deps))
		pr.Get("/transactions/{id}/delete-confirm", deleteConfirmTransactionHandler(deps))
		pr.Patch("/transactions/{id}", updateTransactionHandler(deps))
		pr.Delete("/transactions/{id}", deleteTransactionHandler(deps))
		pr.Get("/dashboard", dashboardPage(deps))
	})

	return r
}

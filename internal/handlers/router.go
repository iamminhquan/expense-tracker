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
		pr.Post("/categories/{id}/delete", deleteCategoryHandler(deps))
		pr.Get("/transactions", transactionsPage(deps))
		pr.Post("/transactions", transactionsPage(deps))
		pr.Post("/transactions/{id}/delete", deleteTransactionHandler(deps))
		pr.Get("/dashboard", dashboardPage(deps))
	})

	return r
}

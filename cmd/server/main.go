package main

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/config"
	"expensetracker/internal/database"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlcgen.New(pool)
	tmpl := template.Must(template.ParseGlob("internal/web/templates/*.html"))

	deps := handlers.Deps{
		DB:         pool,
		Queries:    queries,
		Sessions:   auth.NewManager(queries),
		Templates:  tmpl,
		CookieName: cfg.SessionCookieName,
	}

	router := handlers.NewRouter(deps)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}

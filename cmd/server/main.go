package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/config"
	"expensetracker/internal/database"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// migrationsSourceURL points at the migrations directory relative to the
// process's working directory: repo root for `go run ./cmd/server` locally,
// and /app (the Dockerfile's WORKDIR, which COPYs
// internal/database/migrations to the same relative path) in the container.
const migrationsSourceURL = "file://internal/database/migrations"

// runMigrations applies all pending golang-migrate migrations against dsn
// before the server starts serving requests. Without this, a fresh deploy
// has an empty database and every page fails on its first query.
// migrationsPath is a parameter (rather than the migrationsSourceURL
// constant) so tests can point it at the same migrations directory using a
// path relative to their own package instead of the repo root.
func runMigrations(dsn, migrationsPath string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("new migrate instance: %w", err)
	}
	defer m.Close()

	// migrate.ErrNoChange means the schema was already up to date -- the
	// normal case on every restart after the first deploy -- and is treated
	// as success, not an error.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	if err := runMigrations(cfg.DatabaseURL, migrationsSourceURL); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlcgen.New(pool)
	templates := map[string]*template.Template{
		"auth":         template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/auth.html", "internal/web/templates/auth_card_body.html")),
		"categories":   template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/categories.html", "internal/web/templates/category_row.html")),
		"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/transactions.html")),
		"dashboard":    template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/dashboard.html")),
	}

	deps := handlers.Deps{
		DB:            pool,
		Queries:       queries,
		Sessions:      auth.NewManager(queries),
		Templates:     templates,
		CookieName:    cfg.SessionCookieName,
		SecureCookies: cfg.SecureCookies,
	}

	router := handlers.NewRouter(deps)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}

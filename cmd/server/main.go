package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"expensetracker/internal/auth"
	"expensetracker/internal/config"
	"expensetracker/internal/database"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

// migrationsSourceURL points at the migrations directory relative to the
// process's working directory, i.e. the repo root for `go run ./cmd/server`.
// A deployed binary must therefore be started from a directory that has
// internal/database/migrations at the same relative path.
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
	// .env is optional: a deployment that sets these variables directly
	// (e.g. via the process environment) has no file to load, and that is
	// not an error. Load() also never overrides a variable already set in
	// the environment, so an explicit export still wins over the file.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
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
		"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/transactions.html", "internal/web/templates/transaction_row.html")),
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

package database_test

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// createThrowawayDatabase creates a uniquely-named database on the same
// Postgres server as baseDSN and returns a DSN pointing at it, registering
// a t.Cleanup that drops it afterward.
//
// TestMigrationsApplyCleanly runs m.Up() then m.Down() -- if it ran against
// TEST_DATABASE_URL directly (the same database every other package's tests
// rely on having its schema present), it would race with sibling packages:
// Go runs different packages' test binaries concurrently by default, so
// whichever package happens to be mid-test when this one tears the schema
// down via m.Down() fails, regardless of source-file ordering and even with
// `go test -p 1` (that flag only serializes builds, not the race between
// which package's process is active when). Giving this test its own
// database removes it from the shared state entirely.
func createThrowawayDatabase(t *testing.T, baseDSN string) string {
	t.Helper()

	u, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	dbName := fmt.Sprintf("migrate_test_%d", time.Now().UnixNano())

	adminURL := *u
	adminURL.Path = "/postgres"

	adminDB, err := sql.Open("pgx", adminURL.String())
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
		t.Fatalf("create throwaway database %q: %v", dbName, err)
	}

	t.Cleanup(func() {
		// Can't drop a database while connected to it, so open a fresh
		// connection back to the admin database to run the DROP.
		cleanupDB, err := sql.Open("pgx", adminURL.String())
		if err != nil {
			t.Logf("cleanup: open admin connection: %v", err)
			return
		}
		defer cleanupDB.Close()
		if _, err := cleanupDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, dbName)); err != nil {
			t.Logf("cleanup: drop throwaway database %q: %v", dbName, err)
		}
	})

	throwawayURL := *u
	throwawayURL.Path = "/" + dbName
	return throwawayURL.String()
}

func TestMigrationsApplyCleanly(t *testing.T) {
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set; start docker-compose postgres and export it to run this test")
	}

	dsn := createThrowawayDatabase(t, baseDSN)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("postgres driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		t.Fatalf("new migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT count(*) FROM categories WHERE user_id IS NULL").Scan(&count); err != nil {
		t.Fatalf("query default categories: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected 8 default categories, got %d", count)
	}

	if err := m.Down(); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
}

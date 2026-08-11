package main

import (
	"os"
	"testing"
)

// TestRunMigrations covers Finding 1 from the final whole-branch review:
// nothing ever applied migrations at startup. It exercises the same
// runMigrations function main() calls, against the migrations directory
// relative to this test package (cmd/server), rather than the repo-root
// relative path main() uses -- go test runs with the package directory as
// its working directory, not the repo root.
func TestRunMigrations(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	const migrationsPath = "file://../../internal/database/migrations"

	if err := runMigrations(dsn, migrationsPath); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}

	// Running again against an already-migrated database must not be
	// treated as an error (migrate.ErrNoChange is the normal case on every
	// restart after the first deploy).
	if err := runMigrations(dsn, migrationsPath); err != nil {
		t.Fatalf("runMigrations (second call, already migrated): %v", err)
	}
}

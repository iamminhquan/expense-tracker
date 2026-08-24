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
		t.Skip("TEST_DATABASE_URL not set; point it at a local Postgres and export it to run this test")
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
	// A from-scratch migrate-up has no transactions referencing the legacy
	// "Thu nhập khác" default, so 000006's conditional DELETE removes it,
	// leaving exactly the 9 shared defaults 000006 seeds.
	if count != 9 {
		t.Fatalf("expected 9 default categories, got %d", count)
	}

	var thuNhapKhacCount int
	if err := db.QueryRow(
		"SELECT count(*) FROM categories WHERE user_id IS NULL AND name = 'Thu nhập khác'",
	).Scan(&thuNhapKhacCount); err != nil {
		t.Fatalf("query 'Thu nhập khác' count: %v", err)
	}
	if thuNhapKhacCount != 0 {
		t.Fatalf("expected 'Thu nhập khác' to be gone on a fresh install, found %d rows", thuNhapKhacCount)
	}

	// Addressed by slug rather than by name: 000008 moved the display names
	// to English precisely so nothing has to identify a default category by
	// the string on screen.
	var foodDrinkName, foodDrinkColor string
	if err := db.QueryRow(
		"SELECT name, color FROM categories WHERE slug = 'food_drink'",
	).Scan(&foodDrinkName, &foodDrinkColor); err != nil {
		t.Fatalf("query food_drink category: %v", err)
	}
	if foodDrinkName != "Food & Drink" || foodDrinkColor != "#D97757" {
		t.Fatalf("expected food_drink to be Food & Drink/#D97757, got %q/%q", foodDrinkName, foodDrinkColor)
	}

	// Every default category must carry a slug after 000008; any that does
	// not would render from its name column and be invisible to a future
	// language switcher.
	var slugless int
	if err := db.QueryRow("SELECT count(*) FROM categories WHERE user_id IS NULL AND slug IS NULL").Scan(&slugless); err != nil {
		t.Fatalf("query slugless defaults: %v", err)
	}
	if slugless != 0 {
		t.Fatalf("expected every default category to have a slug, found %d without one", slugless)
	}

	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES (NULL, 'Bad Color Test', 'expense', '#000000')`,
	); err == nil {
		t.Fatal("expected inserting a category with a color outside the fixed palette to violate the CHECK constraint")
	}

	var userID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name, username) VALUES ('migrate-constraint-test@example.com', 'x', 'Constraint Test', 'constraint_test') RETURNING id`,
	).Scan(&userID); err != nil {
		t.Fatalf("insert throwaway user: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Dup', 'expense', '#D97757')`, userID,
	); err != nil {
		t.Fatalf("insert first Dup category: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Dup', 'expense', '#5B8DEF')`, userID,
	); err == nil {
		t.Fatal("expected a second category with the same (user_id, type, name) to violate the unique index")
	}

	if err := m.Down(); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
}

// TestMigrations000006PreservesExistingData exercises the exact scenario the
// rest of this file's tests blind-spot: applying the 000006 redesign
// migration against a database that already has real data referencing the
// pre-redesign default categories (000005's placeholder seed), plus
// custom categories a user created before the redesign shipped. An earlier,
// destructive version of 000006 (DELETE FROM categories WHERE user_id IS
// NULL, then re-insert) would fail here: deleting a default category that a
// transaction still references violates the transactions.category_id FK
// (no ON DELETE clause -- see 000004_create_transactions.up.sql). This test
// proves the rewritten, data-preserving 000006 applies cleanly instead, and
// that every category a pre-existing transaction pointed at is still a
// valid, resolvable row afterward.
func TestMigrations000006PreservesExistingData(t *testing.T) {
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set; point it at a local Postgres and export it to run this test")
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

	// Apply only through 000005 -- the state of the schema right before the
	// 000006 redesign migration this test targets, with the old 8-category
	// placeholder seed already in place.
	if err := m.Steps(5); err != nil {
		t.Fatalf("migrate up through 000005: %v", err)
	}

	// Seed data that mimics a real, already-in-use database: a user, a
	// transaction pointing at one of the old default categories, a custom
	// category with an off-palette color (the free-form picker predates the
	// fixed-palette redesign) with its own transaction, and two custom
	// categories that collide on (user_id, type, name) -- nothing prevented
	// that before 000006 adds the unique index.
	var userID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"preexisting-data@example.com", "x", "Preexisting Data User",
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var oldAnUongID int64
	if err := db.QueryRow(
		`SELECT id FROM categories WHERE user_id IS NULL AND name = 'Ăn uống'`,
	).Scan(&oldAnUongID); err != nil {
		t.Fatalf("query old 'Ăn uống' default: %v", err)
	}

	var oldDefaultTxnID int64
	if err := db.QueryRow(
		`INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
		 VALUES ($1, $2, 50000, 'expense', 'pre-redesign default-category txn', '2026-01-05') RETURNING id`,
		userID, oldAnUongID,
	).Scan(&oldDefaultTxnID); err != nil {
		t.Fatalf("insert transaction on old default category: %v", err)
	}

	// "Thu nhập khác" has no equivalent in the new 9-category set, so 000006
	// conditionally deletes it -- but only when nothing references it. Seed
	// a transaction against it here to exercise the "keep" branch: an
	// upgraded account that already used it must not have its data broken.
	var oldThuNhapKhacID int64
	if err := db.QueryRow(
		`SELECT id FROM categories WHERE user_id IS NULL AND name = 'Thu nhập khác'`,
	).Scan(&oldThuNhapKhacID); err != nil {
		t.Fatalf("query old 'Thu nhập khác' default: %v", err)
	}

	var thuNhapKhacTxnID int64
	if err := db.QueryRow(
		`INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
		 VALUES ($1, $2, 100000, 'income', 'pre-redesign Thu nhap khac txn', '2026-01-05') RETURNING id`,
		userID, oldThuNhapKhacID,
	).Scan(&thuNhapKhacTxnID); err != nil {
		t.Fatalf("insert transaction on old 'Thu nhập khác' default category: %v", err)
	}

	var offPaletteCategoryID int64
	if err := db.QueryRow(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Danh mục cũ', 'expense', '#123456') RETURNING id`,
		userID,
	).Scan(&offPaletteCategoryID); err != nil {
		t.Fatalf("insert off-palette custom category: %v", err)
	}

	var offPaletteTxnID int64
	if err := db.QueryRow(
		`INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
		 VALUES ($1, $2, 20000, 'expense', 'off-palette custom-category txn', '2026-01-06') RETURNING id`,
		userID, offPaletteCategoryID,
	).Scan(&offPaletteTxnID); err != nil {
		t.Fatalf("insert transaction on off-palette custom category: %v", err)
	}

	var dupFirstID, dupSecondID int64
	if err := db.QueryRow(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Trùng tên', 'expense', '#D97757') RETURNING id`,
		userID,
	).Scan(&dupFirstID); err != nil {
		t.Fatalf("insert first duplicate-named category: %v", err)
	}
	if err := db.QueryRow(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Trùng tên', 'expense', '#5B8DEF') RETURNING id`,
		userID,
	).Scan(&dupSecondID); err != nil {
		t.Fatalf("insert second duplicate-named category: %v", err)
	}

	// Apply exactly one step -- 000006, the migration under test. Not m.Up():
	// later migrations rename these same default categories (000008 moves
	// them to English), so running past 000006 would test their end state
	// instead of this one's.
	if err := m.Steps(1); err != nil {
		t.Fatalf("migrate up through 000006: %v", err)
	}

	// The transaction that referenced the old "Ăn uống" default must still
	// resolve to a valid, correctly-named/colored category row.
	var resolvedName, resolvedColor string
	if err := db.QueryRow(
		`SELECT c.name, c.color FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.id = $1`,
		oldDefaultTxnID,
	).Scan(&resolvedName, &resolvedColor); err != nil {
		t.Fatalf("resolve old default category via transaction join: %v", err)
	}
	if resolvedName != "Ăn uống" || resolvedColor != "#D97757" {
		t.Fatalf("expected the pre-existing transaction's category to still resolve to Ăn uống/#D97757, got %q/%q", resolvedName, resolvedColor)
	}

	// The transaction referencing "Thu nhập khác" must still resolve too --
	// the conditional DELETE must not have fired since a transaction
	// references it, and it must have been recolored (not left at its old
	// seed color) so it satisfies the new fixed-palette CHECK constraint.
	var thuNhapKhacName, thuNhapKhacColor string
	if err := db.QueryRow(
		`SELECT c.name, c.color FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.id = $1`,
		thuNhapKhacTxnID,
	).Scan(&thuNhapKhacName, &thuNhapKhacColor); err != nil {
		t.Fatalf("resolve 'Thu nhập khác' category via transaction join: %v", err)
	}
	if thuNhapKhacName != "Thu nhập khác" || thuNhapKhacColor != "#6BA292" {
		t.Fatalf("expected the pre-existing transaction's category to still resolve to Thu nhập khác/#6BA292, got %q/%q", thuNhapKhacName, thuNhapKhacColor)
	}

	// The off-palette custom category must have been recolored into the
	// fixed palette, and its transaction must still point at the same row
	// (id unchanged -- only the color column was touched).
	inPalette := map[string]bool{
		"#D97757": true, "#5B8DEF": true, "#8B7BD8": true, "#6BA292": true,
		"#E0A82E": true, "#D97AA0": true, "#4FA871": true, "#7CA65C": true, "#A1A1AA": true,
	}
	var customColor string
	if err := db.QueryRow(`SELECT color FROM categories WHERE id = $1`, offPaletteCategoryID).Scan(&customColor); err != nil {
		t.Fatalf("query recolored custom category: %v", err)
	}
	if !inPalette[customColor] {
		t.Fatalf("expected off-palette custom category to be reassigned an in-palette color, got %q", customColor)
	}
	var offPaletteTxnCategoryID int64
	if err := db.QueryRow(`SELECT category_id FROM transactions WHERE id = $1`, offPaletteTxnID).Scan(&offPaletteTxnCategoryID); err != nil {
		t.Fatalf("query off-palette transaction: %v", err)
	}
	if offPaletteTxnCategoryID != offPaletteCategoryID {
		t.Fatalf("expected off-palette transaction to still reference category %d, got %d", offPaletteCategoryID, offPaletteTxnCategoryID)
	}

	// The duplicate-named pair must be disambiguated: the older row (lower
	// id) keeps its original name, the newer one gets a numeric suffix.
	var dupFirstName, dupSecondName string
	if err := db.QueryRow(`SELECT name FROM categories WHERE id = $1`, dupFirstID).Scan(&dupFirstName); err != nil {
		t.Fatalf("query first duplicate category name: %v", err)
	}
	if err := db.QueryRow(`SELECT name FROM categories WHERE id = $1`, dupSecondID).Scan(&dupSecondName); err != nil {
		t.Fatalf("query second duplicate category name: %v", err)
	}
	if dupFirstName != "Trùng tên" {
		t.Fatalf("expected the older duplicate to keep its name, got %q", dupFirstName)
	}
	if dupSecondName != "Trùng tên (2)" {
		t.Fatalf("expected the newer duplicate to be suffixed, got %q", dupSecondName)
	}

	// The old 8-category seed (including the kept-but-recolored "Thu nhập
	// khác") plus the 2 genuinely new defaults ("Khác", "Thưởng") should be
	// present: 10 total default rows, none deleted.
	var defaultCount int
	if err := db.QueryRow("SELECT count(*) FROM categories WHERE user_id IS NULL").Scan(&defaultCount); err != nil {
		t.Fatalf("query default category count: %v", err)
	}
	if defaultCount != 10 {
		t.Fatalf("expected 10 default categories after upgrade, got %d", defaultCount)
	}

	// The best-effort down-migration must not error (it's a partial,
	// documented-lossy reverse -- see 000006_redesign_categories.down.sql).
	if err := m.Steps(-1); err != nil {
		t.Fatalf("migrate down one step (000006 down): %v", err)
	}
}

// TestMigrations000008PreservesLegacyOtherIncome covers the one default
// category that a fresh install does not have. 000006 deletes "Thu nhập
// khác" only when no transaction references it, so an account that used it
// still carries the row -- and it is therefore the row most likely to be
// missed when the defaults are renamed. If 000008 skipped it, that account
// would show one Vietnamese category among nine English ones, with no slug
// for a later language switcher to key off.
func TestMigrations000008PreservesLegacyOtherIncome(t *testing.T) {
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set; point it at a local Postgres and export it to run this test")
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

	// Stop at 000005, where "Thu nhập khác" still exists, and give it a
	// transaction so 000006's conditional DELETE has to keep it.
	if err := m.Steps(5); err != nil {
		t.Fatalf("migrate up through 000005: %v", err)
	}

	var userID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"legacy-other-income@example.com", "x", "Legacy Other Income User",
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var legacyID int64
	if err := db.QueryRow(
		`SELECT id FROM categories WHERE user_id IS NULL AND name = 'Thu nhập khác'`,
	).Scan(&legacyID); err != nil {
		t.Fatalf("query legacy 'Thu nhập khác' default: %v", err)
	}

	var txnID int64
	if err := db.QueryRow(
		`INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
		 VALUES ($1, $2, 100000, 'income', 'legacy other-income txn', '2026-01-05') RETURNING id`,
		userID, legacyID,
	).Scan(&txnID); err != nil {
		t.Fatalf("insert transaction on legacy category: %v", err)
	}

	// A custom category, to prove 000008 leaves the user's own words alone.
	var customID int64
	if err := db.QueryRow(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Cà phê', 'expense', '#D97757') RETURNING id`,
		userID,
	).Scan(&customID); err != nil {
		t.Fatalf("insert custom category: %v", err)
	}

	if err := m.Steps(3); err != nil {
		t.Fatalf("migrate up through 000008: %v", err)
	}

	var slug, name string
	if err := db.QueryRow(
		`SELECT c.slug, c.name FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.id = $1`,
		txnID,
	).Scan(&slug, &name); err != nil {
		t.Fatalf("resolve legacy category via transaction join: %v", err)
	}
	if slug != "other_income" || name != "Other income" {
		t.Fatalf("expected the legacy category to become other_income/Other income, got %q/%q", slug, name)
	}

	var customSlug sql.NullString
	var customName string
	if err := db.QueryRow(`SELECT slug, name FROM categories WHERE id = $1`, customID).Scan(&customSlug, &customName); err != nil {
		t.Fatalf("query custom category: %v", err)
	}
	if customSlug.Valid {
		t.Fatalf("expected a user-created category to have no slug, got %q", customSlug.String)
	}
	if customName != "Cà phê" {
		t.Fatalf("expected the user's own category name to be untouched, got %q", customName)
	}

	// Down must put the Vietnamese names back exactly -- unlike 000006's
	// documented-lossy reverse, 000008 records enough to undo itself.
	if err := m.Steps(-1); err != nil {
		t.Fatalf("migrate down one step (000008 down): %v", err)
	}
	var revertedName string
	if err := db.QueryRow(`SELECT name FROM categories WHERE id = $1`, legacyID).Scan(&revertedName); err != nil {
		t.Fatalf("query reverted category: %v", err)
	}
	if revertedName != "Thu nhập khác" {
		t.Fatalf("expected 000008 down to restore 'Thu nhập khác', got %q", revertedName)
	}
}

// TestMigrations000009BackfillsExistingUsers covers 000009's placeholder
// username: applying it against a database that already has user rows (an
// upgraded install, or -- as happens locally -- a shared dev/test database
// with leftover rows from other tests) must not fail the new NOT
// NULL/UNIQUE/CHECK constraints. Each pre-existing row should come out with
// a deterministic 'user_<id>' placeholder that a real user can rename later.
func TestMigrations000009BackfillsExistingUsers(t *testing.T) {
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set; point it at a local Postgres and export it to run this test")
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

	// Stop right before 000009, where users has no username column yet, and
	// seed a row exactly like an upgraded install would already have.
	if err := m.Steps(8); err != nil {
		t.Fatalf("migrate up through 000008: %v", err)
	}

	var userID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`,
		"pre-username@example.com", "x", "Pre Username",
	).Scan(&userID); err != nil {
		t.Fatalf("insert pre-existing user: %v", err)
	}

	if err := m.Steps(1); err != nil {
		t.Fatalf("migrate up through 000009: %v", err)
	}

	var username string
	if err := db.QueryRow(`SELECT username FROM users WHERE id = $1`, userID).Scan(&username); err != nil {
		t.Fatalf("query backfilled username: %v", err)
	}
	if want := fmt.Sprintf("user_%d", userID); username != want {
		t.Fatalf("expected backfilled username %q, got %q", want, username)
	}

	// A second pre-existing row must not collide with the first's
	// placeholder -- the id-derived scheme should keep them unique.
	var secondID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name, username) VALUES ($1, $2, $3, $4) RETURNING id`,
		"post-username@example.com", "x", "Post Username", "post_username",
	).Scan(&secondID); err != nil {
		t.Fatalf("insert post-migration user: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO users (email, password_hash, name, username) VALUES ($1, $2, $3, $4)`,
		"dup-username@example.com", "x", "Dup Username", username,
	); err == nil {
		t.Fatal("expected inserting a duplicate username to violate the unique constraint")
	}

	if _, err := db.Exec(`INSERT INTO users (email, password_hash, name, username) VALUES ($1, $2, $3, $4)`,
		"bad-username@example.com", "x", "Bad Username", "Not_Valid!",
	); err == nil {
		t.Fatal("expected inserting a username that fails the format CHECK to be rejected")
	}

	if err := m.Steps(-1); err != nil {
		t.Fatalf("migrate down one step (000009 down): %v", err)
	}
}

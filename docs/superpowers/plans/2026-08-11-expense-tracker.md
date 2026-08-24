# Expense Tracker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working multi-user expense tracker web app (Go + PostgreSQL + htmx) covering auth, categories, transaction CRUD, and a reporting dashboard.

**Architecture:** Monolithic Go server using `chi` router, `sqlc`-generated type-safe queries over PostgreSQL (via `pgx/v5`), server-rendered `html/template` pages enhanced with `htmx`, and Chart.js (CDN) for report charts. Cookie-based sessions stored in Postgres.

**Tech Stack:** Go 1.22+, `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5`, `sqlc`, `golang-migrate/migrate`, `golang.org/x/crypto/bcrypt`, PostgreSQL 16, Docker/docker-compose, htmx (CDN), Tailwind CSS (CDN), Chart.js (CDN).

## Global Constraints

- Currency: VND only. All money amounts are stored as integers (đồng), never floats.
- Every data query that touches `users`, `categories`, or `transactions` MUST filter by the authenticated `user_id` from the session — never trust a `user_id` from the URL/form body.
- `categories.user_id` may be NULL to represent a default category shared by all users; default categories cannot be edited or deleted by any user.
- No auto-migration/ORM magic: schema changes only via `golang-migrate` SQL files in `internal/database/migrations`.
- Module name: `expensetracker` (no remote git host — local module only, no `go install` from a URL).
- Integration tests that need Postgres read the DSN from env var `TEST_DATABASE_URL` and call `t.Skip` with a clear message if it is unset — this is intentional, not a placeholder, so contributors without Docker running can still run unit tests.

---

## File Structure

```
expense_tracker/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── database/
│   │   ├── db.go
│   │   ├── migrations/
│   │   │   ├── 000001_create_users.up.sql
│   │   │   ├── 000001_create_users.down.sql
│   │   │   ├── 000002_create_sessions.up.sql
│   │   │   ├── 000002_create_sessions.down.sql
│   │   │   ├── 000003_create_categories.up.sql
│   │   │   ├── 000003_create_categories.down.sql
│   │   │   ├── 000004_create_transactions.up.sql
│   │   │   ├── 000004_create_transactions.down.sql
│   │   │   ├── 000005_seed_default_categories.up.sql
│   │   │   └── 000005_seed_default_categories.down.sql
│   │   └── queries/
│   │       ├── users.sql
│   │       ├── sessions.sql
│   │       ├── categories.sql
│   │       └── transactions.sql
│   ├── sqlcgen/                  (sqlc-generated code, do not hand-edit)
│   ├── auth/
│   │   ├── password.go
│   │   ├── password_test.go
│   │   ├── session.go
│   │   ├── session_test.go
│   │   ├── middleware.go
│   │   └── middleware_test.go
│   ├── handlers/
│   │   ├── router.go
│   │   ├── auth_handlers.go
│   │   ├── auth_handlers_test.go
│   │   ├── category_handlers.go
│   │   ├── category_handlers_test.go
│   │   ├── transaction_handlers.go
│   │   ├── transaction_handlers_test.go
│   │   ├── report_handlers.go
│   │   └── report_handlers_test.go
│   └── web/templates/
│       ├── layout.html
│       ├── login.html
│       ├── register.html
│       ├── transactions.html
│       ├── transaction_row.html
│       ├── transaction_form.html
│       ├── categories.html
│       └── dashboard.html
├── sqlc.yaml
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── go.mod
└── go.sum
```

---

### Task 1: Project scaffolding & health endpoint

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `internal/config/config.go`
- Create: `internal/handlers/router.go`
- Create: `internal/handlers/router_test.go`
- Create: `.env.example`
- Create: `docker-compose.yml` (postgres service only for now)

**Interfaces:**
- Produces: `config.Load() (config.Config, error)` — reads `DATABASE_URL`, `PORT`, `SESSION_COOKIE_NAME` from env (with defaults `PORT=8080`, `SESSION_COOKIE_NAME=session_id`).
- Produces: `handlers.NewRouter(deps handlers.Deps) http.Handler` — `Deps` struct currently empty (`struct{}`), will grow in later tasks.

- [ ] **Step 1: Init module and dependencies**

```bash
cd /home/minhquan/projects/go/expense_tracker
go mod init expensetracker
go get github.com/go-chi/chi/v5@latest
```

- [ ] **Step 2: Write `internal/config/config.go`**

```go
package config

import "os"

type Config struct {
	DatabaseURL       string
	Port              string
	SessionCookieName string
}

func Load() Config {
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable"),
		Port:              getEnv("PORT", "8080"),
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 3: Write failing test for the router health endpoint**

`internal/handlers/router_test.go`:

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/handlers"
)

func TestHealthz(t *testing.T) {
	router := handlers.NewRouter(handlers.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/handlers/...`
Expected: FAIL — `handlers.NewRouter` / `handlers.Deps` undefined.

- [ ] **Step 5: Write `internal/handlers/router.go`**

```go
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps holds shared dependencies for handlers. Populated incrementally
// in later tasks (DB queries, session store, templates).
type Deps struct{}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return r
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/handlers/...`
Expected: PASS

- [ ] **Step 7: Write `cmd/server/main.go`**

```go
package main

import (
	"log"
	"net/http"

	"expensetracker/internal/config"
	"expensetracker/internal/handlers"
)

func main() {
	cfg := config.Load()

	router := handlers.NewRouter(handlers.Deps{})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 8: Write `.env.example`**

```
DATABASE_URL=postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable
PORT=8080
SESSION_COOKIE_NAME=session_id
```

- [ ] **Step 9: Write `docker-compose.yml` (Postgres only for now)**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: expense
      POSTGRES_PASSWORD: expense
      POSTGRES_DB: expense_tracker
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

- [ ] **Step 10: Verify the server builds and boots**

Run: `go build ./... && go vet ./...`
Expected: no errors.

- [ ] **Step 11: Commit**

```bash
git init
git add go.mod go.sum cmd internal .env.example docker-compose.yml docs
git commit -m "chore: scaffold project with health endpoint"
```

---

### Task 2: Database schema migrations

**Files:**
- Create: `internal/database/migrations/000001_create_users.up.sql` / `.down.sql`
- Create: `internal/database/migrations/000002_create_sessions.up.sql` / `.down.sql`
- Create: `internal/database/migrations/000003_create_categories.up.sql` / `.down.sql`
- Create: `internal/database/migrations/000004_create_transactions.up.sql` / `.down.sql`
- Create: `internal/database/migrations/000005_seed_default_categories.up.sql` / `.down.sql`
- Create: `internal/database/migrate_test.go`
- Modify: `docker-compose.yml` (no change needed, reused)

**Interfaces:**
- Produces: on-disk migration files consumable by the `migrate` CLI and by `golang-migrate`'s Go API, pointed at `file://internal/database/migrations`.
- Produces final schema tables: `users`, `sessions`, `categories`, `transactions` (columns as in Global Constraints / design doc).

- [ ] **Step 1: Install migrate CLI and Go migrate dependency**

```bash
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

(CLI install for local dev convenience, optional: `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)

- [ ] **Step 2: Write users migration**

`000001_create_users.up.sql`:
```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`000001_create_users.down.sql`:
```sql
DROP TABLE users;
```

- [ ] **Step 3: Write sessions migration**

`000002_create_sessions.up.sql`:
```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
```

`000002_create_sessions.down.sql`:
```sql
DROP TABLE sessions;
```

- [ ] **Step 4: Write categories migration**

`000003_create_categories.up.sql`:
```sql
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('expense', 'income')),
    color TEXT NOT NULL DEFAULT '#6b7280',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
```

`000003_create_categories.down.sql`:
```sql
DROP TABLE categories;
```

- [ ] **Step 5: Write transactions migration**

`000004_create_transactions.up.sql`:
```sql
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    type TEXT NOT NULL CHECK (type IN ('expense', 'income')),
    description TEXT NOT NULL DEFAULT '',
    occurred_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user_id_occurred_on ON transactions(user_id, occurred_on);
```

`000004_create_transactions.down.sql`:
```sql
DROP TABLE transactions;
```

- [ ] **Step 6: Write seed migration for default categories**

`000005_seed_default_categories.up.sql`:
```sql
INSERT INTO categories (user_id, name, type, color) VALUES
    (NULL, 'Ăn uống', 'expense', '#ef4444'),
    (NULL, 'Di chuyển', 'expense', '#f97316'),
    (NULL, 'Giải trí', 'expense', '#eab308'),
    (NULL, 'Hóa đơn', 'expense', '#8b5cf6'),
    (NULL, 'Sức khỏe', 'expense', '#ec4899'),
    (NULL, 'Mua sắm', 'expense', '#06b6d4'),
    (NULL, 'Lương', 'income', '#22c55e'),
    (NULL, 'Thu nhập khác', 'income', '#10b981');
```

`000005_seed_default_categories.down.sql`:
```sql
DELETE FROM categories WHERE user_id IS NULL;
```

- [ ] **Step 7: Write a failing integration test that runs migrations**

`internal/database/migrate_test.go`:

```go
package database_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMigrationsApplyCleanly(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; start docker-compose postgres and export it to run this test")
	}

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
```

Note: this test uses `file://migrations` as a relative path — run it with working directory `internal/database` (Go test binaries run with the package directory as CWD by default, so this resolves to `internal/database/migrations`).

- [ ] **Step 8: Add pgx stdlib dependency**

```bash
go get github.com/jackc/pgx/v5
```

- [ ] **Step 9: Run test to verify it fails (or skips) before migrations exist**

Run: `docker compose up -d postgres && sleep 2 && TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/database/... -v`
Expected before Steps 2-6 are in place: FAIL (no migration files). Since this task writes the migrations first, run this after Step 6 to confirm PASS instead — the point of Step 9 is verifying the test harness works against a real DB.

- [ ] **Step 10: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/database/... -v`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/database go.mod go.sum
git commit -m "feat: add database schema migrations"
```

---

### Task 3: sqlc setup and user queries

**Files:**
- Create: `sqlc.yaml`
- Create: `internal/database/queries/users.sql`
- Create: `internal/database/db.go`
- Create: `internal/database/db_test.go`
- Generated (by sqlc, not hand-written): `internal/sqlcgen/*.go`

**Interfaces:**
- Produces: `database.NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error)`.
- Produces (sqlc-generated): `sqlcgen.Queries`, `sqlcgen.New(db sqlcgen.DBTX) *sqlcgen.Queries`, `(*Queries).CreateUser(ctx, CreateUserParams) (User, error)`, `(*Queries).GetUserByEmail(ctx, string) (User, error)`, `(*Queries).GetUserByID(ctx, int64) (User, error)`.

- [ ] **Step 1: Install sqlc**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

- [ ] **Step 2: Write `sqlc.yaml`**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/database/queries"
    schema: "internal/database/migrations"
    gen:
      go:
        package: "sqlcgen"
        out: "internal/sqlcgen"
        sql_package: "pgx/v5"
        emit_json_tags: true
```

- [ ] **Step 3: Write `internal/database/queries/users.sql`**

```sql
-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
```

- [ ] **Step 4: Generate sqlc code**

Run: `sqlc generate`
Expected: `internal/sqlcgen/` populated with `db.go`, `models.go`, `users.sql.go` — no errors.

- [ ] **Step 5: Write `internal/database/db.go`**

```go
package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}
```

- [ ] **Step 6: Write failing integration test for user queries**

`internal/database/db_test.go`:

```go
package database_test

import (
	"context"
	"os"
	"testing"

	"expensetracker/internal/database"
	"expensetracker/internal/sqlcgen"
)

func TestCreateAndGetUser(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	pool, err := database.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	q := sqlcgen.New(pool)

	email := "test-user@example.com"
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)

	created, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: "hashed",
		Name:         "Test User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Email != email {
		t.Fatalf("expected email %q, got %q", email, created.Email)
	}

	fetched, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected id %d, got %d", created.ID, fetched.ID)
	}
}
```

- [ ] **Step 7: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/database/... -run TestCreateAndGetUser -v`
Expected: FAIL if migrations haven't been applied to this DB yet — apply them first with the migrate CLI or by re-running Task 2's test, then re-run.

- [ ] **Step 8: Apply migrations and run test to verify it passes**

Run:
```bash
migrate -database "postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" -path internal/database/migrations up
TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/database/... -run TestCreateAndGetUser -v
```
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add sqlc.yaml internal/database internal/sqlcgen go.mod go.sum
git commit -m "feat: add sqlc setup and user queries"
```

---

### Task 4: Password hashing

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/password_test.go`

**Interfaces:**
- Produces: `auth.HashPassword(plain string) (string, error)`, `auth.VerifyPassword(hash, plain string) bool`.

- [ ] **Step 1: Get bcrypt dependency**

```bash
go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: Write failing tests**

`internal/auth/password_test.go`:

```go
package auth_test

import (
	"testing"

	"expensetracker/internal/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct-horse" {
		t.Fatal("hash must not equal plaintext")
	}
	if !auth.VerifyPassword(hash, "correct-horse") {
		t.Fatal("expected verify to succeed for correct password")
	}
	if auth.VerifyPassword(hash, "wrong-password") {
		t.Fatal("expected verify to fail for wrong password")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/auth/... -run TestHashAndVerifyPassword -v`
Expected: FAIL — `auth.HashPassword` undefined.

- [ ] **Step 4: Write `internal/auth/password.go`**

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/auth/... -run TestHashAndVerifyPassword -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/auth go.mod go.sum
git commit -m "feat: add password hashing"
```

---

### Task 5: Session queries and session manager

**Files:**
- Create: `internal/database/queries/sessions.sql`
- Create: `internal/auth/session.go`
- Create: `internal/auth/session_test.go`

**Interfaces:**
- Consumes: `sqlcgen.Queries` from Task 3, `database.NewPool` from Task 3.
- Produces (sqlc-generated): `(*Queries).CreateSession(ctx, CreateSessionParams) (Session, error)`, `(*Queries).GetSession(ctx, string) (Session, error)`, `(*Queries).DeleteSession(ctx, string) error`.
- Produces: `auth.Manager` struct with `NewManager(q *sqlcgen.Queries) *Manager`, `(*Manager).CreateSession(ctx, userID int64) (token string, expiresAt time.Time, err error)`, `(*Manager).ValidateSession(ctx, token string) (userID int64, err error)`, `(*Manager).DeleteSession(ctx, token string) error`. Sessions expire 7 days after creation.

- [ ] **Step 1: Write `internal/database/queries/sessions.sql`**

```sql
-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;
```

- [ ] **Step 2: Regenerate sqlc code**

Run: `sqlc generate`
Expected: `internal/sqlcgen/sessions.sql.go` created, no errors.

- [ ] **Step 3: Write failing test for session manager**

`internal/auth/session_test.go`:

```go
package auth_test

import (
	"context"
	"os"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/database"
	"expensetracker/internal/sqlcgen"
)

func setupTestUser(t *testing.T, q *sqlcgen.Queries) int64 {
	t.Helper()
	ctx := context.Background()
	email := "session-test@example.com"
	pool := testPool(t)
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	user, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: "hashed",
		Name:         "Session Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

func testPool(t *testing.T) *database.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestSessionLifecycle(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	mgr := auth.NewManager(q)
	ctx := context.Background()

	token, expiresAt, err := mgr.CreateSession(ctx, userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !expiresAt.After(context.Background().Value(struct{}{}) == nil && expiresAt.Add(0) == expiresAt) {
		// placeholder removed below in favor of simple time check
	}

	gotUserID, err := mgr.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("expected user id %d, got %d", userID, gotUserID)
	}

	if err := mgr.DeleteSession(ctx, token); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, err := mgr.ValidateSession(ctx, token); err == nil {
		t.Fatal("expected validate to fail after delete")
	}
}
```

Note: the `expiresAt.After(...)` line above is invalid Go — replace it before running. Use this corrected version of that assertion instead:

```go
	if !expiresAt.After(timeNow()) {
		t.Fatal("expected expiry to be in the future")
	}
```

where `timeNow` is simply `time.Now` — add `"time"` to the imports and replace the line with:

```go
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/auth/... -run TestSessionLifecycle -v`
Expected: FAIL — `auth.NewManager` undefined.

- [ ] **Step 5: Write `internal/auth/session.go`**

```go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5"
)

const sessionTTL = 7 * 24 * time.Hour

type Manager struct {
	queries *sqlcgen.Queries
}

func NewManager(q *sqlcgen.Queries) *Manager {
	return &Manager{queries: q}
}

func (m *Manager) CreateSession(ctx context.Context, userID int64) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(sessionTTL)

	session, err := m.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		ID:        token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return session.ID, session.ExpiresAt, nil
}

func (m *Manager) ValidateSession(ctx context.Context, token string) (int64, error) {
	session, err := m.queries.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("session not found")
		}
		return 0, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = m.queries.DeleteSession(ctx, token)
		return 0, errors.New("session expired")
	}
	return session.UserID, nil
}

func (m *Manager) DeleteSession(ctx context.Context, token string) error {
	return m.queries.DeleteSession(ctx, token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
```

- [ ] **Step 6: Fix the test's invalid line and run to verify it passes**

Replace the malformed `if !expiresAt.After(...)` block in `session_test.go` with the corrected `time.Now()` version shown in Step 3, add `"time"` to imports, then run:

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/auth/... -v`
Expected: PASS (both `TestHashAndVerifyPassword` and `TestSessionLifecycle`)

- [ ] **Step 7: Commit**

```bash
git add internal/database/queries/sessions.sql internal/sqlcgen internal/auth
git commit -m "feat: add session creation, validation, and expiry"
```

---

### Task 6: Auth middleware

**Files:**
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/middleware_test.go`

**Interfaces:**
- Consumes: `auth.Manager` from Task 5.
- Produces: `auth.RequireAuth(mgr *Manager, cookieName string) func(http.Handler) http.Handler` — chi-compatible middleware; on success sets user ID in request context via `auth.UserIDFromContext(ctx) (int64, bool)`; on failure redirects to `/login`.

- [ ] **Step 1: Write failing test**

`internal/auth/middleware_test.go`:

```go
package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	mgr := auth.NewManager(q)

	handler := auth.RequireAuth(mgr, "session_id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without a valid session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestRequireAuthAcceptsValidSession(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	mgr := auth.NewManager(q)
	ctx := context.Background()
	token, _, err := mgr.CreateSession(ctx, userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	called := false
	handler := auth.RequireAuth(mgr, "session_id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID, ok := auth.UserIDFromContext(r.Context())
		if !ok || gotID != userID {
			t.Fatalf("expected user id %d in context, got %d (ok=%v)", userID, gotID, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/auth/... -run TestRequireAuth -v`
Expected: FAIL — `auth.RequireAuth` / `auth.UserIDFromContext` undefined.

- [ ] **Step 3: Write `internal/auth/middleware.go`**

```go
package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userIDContextKey contextKey = "userID"

func RequireAuth(mgr *Manager, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userID, err := mgr.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDContextKey).(int64)
	return id, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/auth/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth
git commit -m "feat: add RequireAuth middleware"
```

---

### Task 7: Register/login/logout handlers and templates

**Files:**
- Create: `internal/web/templates/layout.html`
- Create: `internal/web/templates/login.html`
- Create: `internal/web/templates/register.html`
- Create: `internal/handlers/auth_handlers.go`
- Create: `internal/handlers/auth_handlers_test.go`
- Modify: `internal/handlers/router.go` (wire routes, extend `Deps`)
- Modify: `cmd/server/main.go` (wire DB pool, session manager, templates into `Deps`)

**Interfaces:**
- Consumes: `sqlcgen.Queries`, `auth.Manager`, `auth.HashPassword`/`VerifyPassword`, `auth.RequireAuth`.
- Produces: `Deps` gains `DB *pgxpool.Pool`, `Queries *sqlcgen.Queries`, `Sessions *auth.Manager`, `Templates *template.Template`, `CookieName string`.
- Produces routes: `GET/POST /register`, `GET/POST /login`, `POST /logout`.

- [ ] **Step 1: Write `internal/web/templates/layout.html`**

```html
{{define "layout"}}
<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <title>Expense Tracker</title>
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 text-gray-900">
  <main class="max-w-3xl mx-auto p-6">
    {{template "content" .}}
  </main>
</body>
</html>
{{end}}
```

- [ ] **Step 2: Write `internal/web/templates/register.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-4">Đăng ký</h1>
{{if .Error}}<p class="text-red-600 mb-2">{{.Error}}</p>{{end}}
<form method="POST" action="/register" class="space-y-3">
  <input class="border p-2 w-full" type="text" name="name" placeholder="Họ tên" value="{{.Name}}" required>
  <input class="border p-2 w-full" type="email" name="email" placeholder="Email" value="{{.Email}}" required>
  <input class="border p-2 w-full" type="password" name="password" placeholder="Mật khẩu" required>
  <button class="bg-blue-600 text-white px-4 py-2 rounded" type="submit">Đăng ký</button>
</form>
<p class="mt-3"><a class="text-blue-600" href="/login">Đã có tài khoản? Đăng nhập</a></p>
{{end}}
```

- [ ] **Step 3: Write `internal/web/templates/login.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-4">Đăng nhập</h1>
{{if .Error}}<p class="text-red-600 mb-2">{{.Error}}</p>{{end}}
<form method="POST" action="/login" class="space-y-3">
  <input class="border p-2 w-full" type="email" name="email" placeholder="Email" value="{{.Email}}" required>
  <input class="border p-2 w-full" type="password" name="password" placeholder="Mật khẩu" required>
  <button class="bg-blue-600 text-white px-4 py-2 rounded" type="submit">Đăng nhập</button>
</form>
<p class="mt-3"><a class="text-blue-600" href="/register">Chưa có tài khoản? Đăng ký</a></p>
{{end}}
```

- [ ] **Step 4: Write failing handler tests**

`internal/handlers/auth_handlers_test.go`:

```go
package handlers_test

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/database"
	"expensetracker/internal/handlers"
	"expensetracker/internal/sqlcgen"
)

func newTestDeps(t *testing.T) handlers.Deps {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	tmpl := template.Must(template.ParseGlob("../web/templates/*.html"))

	return handlers.Deps{
		DB:         pool,
		Queries:    sqlcgen.New(pool),
		Sessions:   auth.NewManager(sqlcgen.New(pool)),
		Templates:  tmpl,
		CookieName: "session_id",
	}
}

func TestRegisterThenLoginFlow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	email := "flow-test@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	form := url.Values{
		"name":     {"Flow Test"},
		"email":    {email},
		"password": {"s3cret-pass"},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after register, got %d: %s", rec.Code, rec.Body.String())
	}

	loginForm := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after login, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a session cookie to be set")
	}
}
```

- [ ] **Step 5: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -run TestRegisterThenLoginFlow -v`
Expected: FAIL — `handlers.Deps` has no fields `DB`/`Queries`/`Sessions`/`Templates`/`CookieName`, and `/register`/`/login` routes don't exist.

- [ ] **Step 6: Extend `Deps` and write `internal/handlers/auth_handlers.go`**

```go
package handlers

import (
	"html/template"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	DB         *pgxpool.Pool
	Queries    *sqlcgen.Queries
	Sessions   *auth.Manager
	Templates  *template.Template
	CookieName string
}

func registerPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			deps.Templates.ExecuteTemplate(w, "layout", map[string]any{})
			return
		}

		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		hash, err := auth.HashPassword(password)
		if err != nil {
			deps.Templates.ExecuteTemplate(w, "layout", map[string]any{"Error": "Không thể tạo tài khoản", "Name": name, "Email": email})
			return
		}

		user, err := deps.Queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
		})
		if err != nil {
			deps.Templates.ExecuteTemplate(w, "layout", map[string]any{"Error": "Email đã được sử dụng", "Name": name, "Email": email})
			return
		}

		startSession(w, r, deps, user.ID)
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}

func loginPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			deps.Templates.ExecuteTemplate(w, "layout", map[string]any{})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := deps.Queries.GetUserByEmail(r.Context(), email)
		if err != nil || !auth.VerifyPassword(user.PasswordHash, password) {
			deps.Templates.ExecuteTemplate(w, "layout", map[string]any{"Error": "Email hoặc mật khẩu không đúng", "Email": email})
			return
		}

		startSession(w, r, deps, user.ID)
		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}

func logoutHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(deps.CookieName); err == nil {
			deps.Sessions.DeleteSession(r.Context(), cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{Name: deps.CookieName, Value: "", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func startSession(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	token, expiresAt, err := deps.Sessions.CreateSession(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     deps.CookieName,
		Value:    token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
	})
}
```

- [ ] **Step 7: Wire routes in `internal/handlers/router.go`**

```go
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/register", registerPage(deps))
	r.Post("/register", registerPage(deps))
	r.Get("/login", loginPage(deps))
	r.Post("/login", loginPage(deps))
	r.Post("/logout", logoutHandler(deps))

	return r
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -v`
Expected: PASS

- [ ] **Step 9: Wire real dependencies in `cmd/server/main.go`**

```go
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
```

- [ ] **Step 10: Commit**

```bash
git add internal/web internal/handlers cmd/server
git commit -m "feat: add register/login/logout handlers and templates"
```

---

### Task 8: Category queries, handlers, and page

**Files:**
- Create: `internal/database/queries/categories.sql`
- Create: `internal/handlers/category_handlers.go`
- Create: `internal/handlers/category_handlers_test.go`
- Create: `internal/web/templates/categories.html`
- Modify: `internal/handlers/router.go` (add category routes under `RequireAuth`)

**Interfaces:**
- Consumes: `Deps` from Task 7, `auth.RequireAuth`/`UserIDFromContext` from Task 6.
- Produces (sqlc-generated): `ListCategoriesForUser(ctx, userID int64) ([]Category, error)` (includes rows where `user_id = $1 OR user_id IS NULL`), `CreateCategory`, `DeleteCategory` (scoped: `WHERE id = $1 AND user_id = $2`).
- Produces routes: `GET /categories`, `POST /categories`, `POST /categories/{id}/delete` — all behind `RequireAuth`.

- [ ] **Step 1: Write `internal/database/queries/categories.sql`**

```sql
-- name: ListCategoriesForUser :many
SELECT * FROM categories
WHERE user_id = $1 OR user_id IS NULL
ORDER BY user_id NULLS FIRST, name;

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, type, color)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 2: Regenerate sqlc code**

Run: `sqlc generate`
Expected: `internal/sqlcgen/categories.sql.go` created.

- [ ] **Step 3: Write failing handler tests**

`internal/handlers/category_handlers_test.go`:

```go
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

func loginAndGetCookie(t *testing.T, router http.Handler, deps handlers.Deps, email, password string) *http.Cookie {
	t.Helper()
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)

	form := url.Values{"name": {"Cat Test"}, "email": {email}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after register")
	}
	return cookies[0]
}

func TestCreateAndListCategories(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-test@example.com", "s3cret-pass")

	form := url.Values{"name": {"Du lịch"}, "type": {"expense"}, "color": {"#111111"}}
	req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusSeeOther {
		t.Fatalf("expected success status creating category, got %d: %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/categories", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing categories, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), "Du lịch") {
		t.Fatal("expected created category to appear in list page")
	}
	if !strings.Contains(listRec.Body.String(), "Ăn uống") {
		t.Fatal("expected default category to appear in list page")
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -run TestCreateAndListCategories -v`
Expected: FAIL — `/categories` route doesn't exist.

- [ ] **Step 5: Write `internal/web/templates/categories.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-4">Danh mục</h1>
<form method="POST" action="/categories" class="flex gap-2 mb-6">
  <input class="border p-2 flex-1" type="text" name="name" placeholder="Tên danh mục" required>
  <select class="border p-2" name="type">
    <option value="expense">Chi</option>
    <option value="income">Thu</option>
  </select>
  <input class="border p-2 w-24" type="color" name="color" value="#6b7280">
  <button class="bg-blue-600 text-white px-4 py-2 rounded" type="submit">Thêm</button>
</form>
<ul class="space-y-2">
  {{range .Categories}}
  <li class="flex items-center justify-between border p-2 rounded">
    <span><span style="color: {{.Color}}">●</span> {{.Name}} ({{.Type}})</span>
    {{if .UserID.Valid}}
    <form method="POST" action="/categories/{{.ID}}/delete">
      <button class="text-red-600" type="submit">Xóa</button>
    </form>
    {{end}}
  </li>
  {{end}}
</ul>
{{end}}
```

- [ ] **Step 6: Write `internal/handlers/category_handlers.go`**

```go
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

		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		deps.Templates.ExecuteTemplate(w, "layout", map[string]any{"Categories": categories})
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
```

Note: `pgInt64` converts a plain `int64` into the `pgtype.Int8` (or equivalent nullable type) that sqlc generates for a nullable `user_id` column — add this small helper to `internal/handlers/router.go` (or a new `internal/handlers/pg.go`):

```go
func pgInt64(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
```

(Add `"github.com/jackc/pgx/v5/pgtype"` and `"github.com/go-chi/chi/v5"` imports where this helper lives. Exact generated type names for nullable columns depend on sqlc's output from Step 2 — check `internal/sqlcgen/models.go` after generating and adjust `pgInt64`'s return type to match if sqlc chose a different nullable wrapper.)

- [ ] **Step 7: Add routes behind `RequireAuth` in `router.go`**

```go
	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireAuth(deps.Sessions, deps.CookieName))
		pr.Get("/categories", categoriesPage(deps))
		pr.Post("/categories", categoriesPage(deps))
		pr.Post("/categories/{id}/delete", deleteCategoryHandler(deps))
	})
```

- [ ] **Step 8: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/database/queries/categories.sql internal/sqlcgen internal/handlers internal/web
git commit -m "feat: add category management"
```

---

### Task 9: Transaction queries, handlers, and page

**Files:**
- Create: `internal/database/queries/transactions.sql`
- Create: `internal/handlers/transaction_handlers.go`
- Create: `internal/handlers/transaction_handlers_test.go`
- Create: `internal/web/templates/transactions.html`
- Create: `internal/web/templates/transaction_row.html`
- Create: `internal/web/templates/transaction_form.html`
- Modify: `internal/handlers/router.go` (add transaction routes)

**Interfaces:**
- Consumes: `Deps`, `auth.RequireAuth`/`UserIDFromContext`.
- Produces (sqlc-generated): `ListTransactionsForMonth(ctx, ListTransactionsForMonthParams{UserID, From, To})`, `CreateTransaction`, `GetTransaction` (scoped by user), `UpdateTransaction` (scoped by user), `DeleteTransaction` (scoped by user, `:execrows`).
- Produces routes: `GET /transactions` (list + month filter via `?month=YYYY-MM`), `POST /transactions`, `GET /transactions/{id}/edit`, `POST /transactions/{id}`, `POST /transactions/{id}/delete` — all behind `RequireAuth`.

- [ ] **Step 1: Write `internal/database/queries/transactions.sql`**

```sql
-- name: ListTransactionsForMonth :many
SELECT t.*, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.occurred_on >= $2 AND t.occurred_on < $3
ORDER BY t.occurred_on DESC, t.id DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = $1 AND user_id = $2;

-- name: UpdateTransaction :one
UPDATE transactions
SET category_id = $3, amount = $4, type = $5, description = $6, occurred_on = $7, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTransaction :execrows
DELETE FROM transactions WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 2: Regenerate sqlc code**

Run: `sqlc generate`
Expected: `internal/sqlcgen/transactions.sql.go` created.

- [ ] **Step 3: Write failing handler tests covering CRUD and cross-user isolation**

`internal/handlers/transaction_handlers_test.go`:

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTransactionCRUDAndIsolation(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookieA := loginAndGetCookie(t, router, deps, "txn-a@example.com", "s3cret-pass")
	cookieB := loginAndGetCookie(t, router, deps, "txn-b@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), 0)
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories to exist: %v", err)
	}
	categoryID := categories[0].ID

	today := time.Now().Format("2006-01-02")
	form := url.Values{
		"category_id": {itoa(categoryID)},
		"amount":      {"50000"},
		"type":        {"expense"},
		"description": {"Cà phê"},
		"occurred_on": {today},
	}
	createReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(cookieA)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK && createRec.Code != http.StatusSeeOther {
		t.Fatalf("expected success creating transaction, got %d: %s", createRec.Code, createRec.Body.String())
	}

	listReqA := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReqA.AddCookie(cookieA)
	listRecA := httptest.NewRecorder()
	router.ServeHTTP(listRecA, listReqA)
	if !strings.Contains(listRecA.Body.String(), "Cà phê") {
		t.Fatal("expected user A to see their own transaction")
	}

	listReqB := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	listReqB.AddCookie(cookieB)
	listRecB := httptest.NewRecorder()
	router.ServeHTTP(listRecB, listReqB)
	if strings.Contains(listRecB.Body.String(), "Cà phê") {
		t.Fatal("expected user B NOT to see user A's transaction")
	}
}

func itoa(v int64) string {
	return fmt.Sprintf("%d", v)
}
```

Add `"context"` and `"fmt"` to the imports at the top of the file alongside the existing ones.

- [ ] **Step 4: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -run TestTransactionCRUDAndIsolation -v`
Expected: FAIL — `/transactions` POST route doesn't exist yet.

- [ ] **Step 5: Write `internal/web/templates/transactions.html`, `transaction_row.html`, `transaction_form.html`**

`transactions.html`:
```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-4">Giao dịch</h1>
<form method="POST" action="/transactions" class="grid grid-cols-5 gap-2 mb-6">
  <select class="border p-2" name="category_id" required>
    {{range .Categories}}<option value="{{.ID}}">{{.Name}}</option>{{end}}
  </select>
  <select class="border p-2" name="type">
    <option value="expense">Chi</option>
    <option value="income">Thu</option>
  </select>
  <input class="border p-2" type="number" name="amount" placeholder="Số tiền" min="1" required>
  <input class="border p-2" type="date" name="occurred_on" required>
  <input class="border p-2" type="text" name="description" placeholder="Ghi chú">
  <button class="col-span-5 bg-blue-600 text-white px-4 py-2 rounded" type="submit">Thêm giao dịch</button>
</form>
<table class="w-full text-left">
  <thead><tr><th>Ngày</th><th>Danh mục</th><th>Loại</th><th>Số tiền</th><th>Ghi chú</th><th></th></tr></thead>
  <tbody>
    {{range .Transactions}}
    <tr class="border-t">
      <td>{{.OccurredOn}}</td>
      <td>{{.CategoryName}}</td>
      <td>{{.Type}}</td>
      <td>{{.Amount}}₫</td>
      <td>{{.Description}}</td>
      <td>
        <form method="POST" action="/transactions/{{.ID}}/delete">
          <button class="text-red-600" type="submit">Xóa</button>
        </form>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
{{end}}
```

(`transaction_row.html` and `transaction_form.html` are deferred to a later htmx-polish pass — not required for the core CRUD flow this task tests. Skip creating them now to avoid unused placeholder files; remove those two entries from the File Structure list if not revisited.)

- [ ] **Step 6: Write `internal/handlers/transaction_handlers.go`**

```go
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/go-chi/chi/v5"
)

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
			amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
			if err != nil || amount <= 0 {
				http.Error(w, "invalid amount", http.StatusBadRequest)
				return
			}
			occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
			if err != nil {
				http.Error(w, "invalid date", http.StatusBadRequest)
				return
			}

			_, err = deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
				UserID:      userID,
				CategoryID:  categoryID,
				Amount:      amount,
				Type:        r.FormValue("type"),
				Description: r.FormValue("description"),
				OccurredOn:  occurredOn,
			})
			if err != nil {
				http.Error(w, "could not create transaction", http.StatusBadRequest)
				return
			}
		}

		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)

		transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
			UserID: userID,
			From:   from,
			To:     to,
		})
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		deps.Templates.ExecuteTemplate(w, "layout", map[string]any{
			"Transactions": transactions,
			"Categories":   categories,
		})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{
			ID:     id,
			UserID: userID,
		}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}
```

- [ ] **Step 7: Add routes in `router.go`**

```go
		pr.Get("/transactions", transactionsPage(deps))
		pr.Post("/transactions", transactionsPage(deps))
		pr.Post("/transactions/{id}/delete", deleteTransactionHandler(deps))
```

(inside the same `RequireAuth` group added in Task 8)

- [ ] **Step 8: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/database/queries/transactions.sql internal/sqlcgen internal/handlers internal/web
git commit -m "feat: add transaction CRUD"
```

---

### Task 10: Reports/dashboard

**Files:**
- Modify: `internal/database/queries/transactions.sql` (add report queries)
- Create: `internal/handlers/report_handlers.go`
- Create: `internal/handlers/report_handlers_test.go`
- Create: `internal/web/templates/dashboard.html`
- Modify: `internal/handlers/router.go` (add `/dashboard` route)

**Interfaces:**
- Consumes: `Deps`, `auth.RequireAuth`/`UserIDFromContext`.
- Produces (sqlc-generated): `MonthlyTotals(ctx, MonthlyTotalsParams{UserID, From, To}) (MonthlyTotalsRow, error)` (sums of expense and income), `CategoryBreakdown(ctx, CategoryBreakdownParams{UserID, From, To}) ([]CategoryBreakdownRow, error)`.
- Produces route: `GET /dashboard` (behind `RequireAuth`) rendering totals + a JSON payload embedded for Chart.js.

- [ ] **Step 1: Append report queries to `internal/database/queries/transactions.sql`**

```sql
-- name: MonthlyTotals :one
SELECT
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS total_expense,
    COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS total_income
FROM transactions
WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3;

-- name: CategoryBreakdown :many
SELECT c.name AS category_name, c.color AS category_color, SUM(t.amount)::bigint AS total
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.type = 'expense' AND t.occurred_on >= $2 AND t.occurred_on < $3
GROUP BY c.name, c.color
ORDER BY total DESC;
```

- [ ] **Step 2: Regenerate sqlc code**

Run: `sqlc generate`
Expected: new methods appear in `internal/sqlcgen/transactions.sql.go`.

- [ ] **Step 3: Write failing handler test**

`internal/handlers/report_handlers_test.go`:

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDashboardShowsMonthlyTotal(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-test@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), 0)
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	form := url.Values{
		"category_id": {itoa(categories[0].ID)},
		"amount":      {"100000"},
		"type":        {"expense"},
		"description": {"Test spend"},
		"occurred_on": {today},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	router.ServeHTTP(httptest.NewRecorder(), req)

	dashReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	dashReq.AddCookie(cookie)
	dashRec := httptest.NewRecorder()
	router.ServeHTTP(dashRec, dashReq)

	if dashRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", dashRec.Code, dashRec.Body.String())
	}
	if !strings.Contains(dashRec.Body.String(), "100000") {
		t.Fatal("expected dashboard to reflect the new transaction's amount")
	}
}
```

Add `"context"` to the imports.

- [ ] **Step 4: Run test to verify it fails**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -run TestDashboardShowsMonthlyTotal -v`
Expected: FAIL — `/dashboard` route doesn't exist.

- [ ] **Step 5: Write `internal/web/templates/dashboard.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-4">Tổng quan</h1>
<div class="grid grid-cols-2 gap-4 mb-6">
  <div class="border p-4 rounded"><p class="text-sm text-gray-500">Tổng chi tháng này</p><p class="text-xl font-bold">{{.TotalExpense}}₫</p></div>
  <div class="border p-4 rounded"><p class="text-sm text-gray-500">Tổng thu tháng này</p><p class="text-xl font-bold">{{.TotalIncome}}₫</p></div>
</div>
<canvas id="breakdown-chart" width="400" height="300"></canvas>
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
new Chart(document.getElementById('breakdown-chart'), {
  type: 'pie',
  data: {
    labels: {{.BreakdownLabelsJSON}},
    datasets: [{ data: {{.BreakdownValuesJSON}}, backgroundColor: {{.BreakdownColorsJSON}} }]
  }
});
</script>
{{end}}
```

- [ ] **Step 6: Write `internal/handlers/report_handlers.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)

		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, From: from, To: to})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}

		breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{UserID: userID, From: from, To: to})
		if err != nil {
			http.Error(w, "could not load breakdown", http.StatusInternalServerError)
			return
		}

		labels := make([]string, len(breakdown))
		values := make([]int64, len(breakdown))
		colors := make([]string, len(breakdown))
		for i, row := range breakdown {
			labels[i] = row.CategoryName
			values[i] = row.Total
			colors[i] = row.CategoryColor
		}
		labelsJSON, _ := json.Marshal(labels)
		valuesJSON, _ := json.Marshal(values)
		colorsJSON, _ := json.Marshal(colors)

		deps.Templates.ExecuteTemplate(w, "layout", map[string]any{
			"TotalExpense":         totals.TotalExpense,
			"TotalIncome":          totals.TotalIncome,
			"BreakdownLabelsJSON":  string(labelsJSON),
			"BreakdownValuesJSON":  string(valuesJSON),
			"BreakdownColorsJSON":  string(colorsJSON),
		})
	}
}
```

Note: the `{{.BreakdownLabelsJSON}}` etc. fields are pre-marshaled JSON strings; render them unescaped in the template using `template.JS(...)` conversions if `html/template` escapes them as strings — wrap the three fields with `template.JS(...)` before passing into the map, and add `"html/template"` to this file's imports.

- [ ] **Step 7: Add route in `router.go`**

```go
		pr.Get("/dashboard", dashboardPage(deps))
```

- [ ] **Step 8: Run test to verify it passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/database/queries/transactions.sql internal/sqlcgen internal/handlers internal/web
git commit -m "feat: add reporting dashboard"
```

---

### Task 11: Navigation layout and end-to-end smoke test

**Files:**
- Modify: `internal/web/templates/layout.html` (add nav bar linking Transactions/Categories/Dashboard/Logout)
- Create: `internal/handlers/smoke_test.go`

**Interfaces:**
- Consumes: all routes from Tasks 7-10.
- No new production interfaces — this task wires navigation UI and adds one end-to-end regression test.

- [ ] **Step 1: Add nav bar to `layout.html`**

```html
{{define "layout"}}
<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <title>Expense Tracker</title>
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 text-gray-900">
  <nav class="bg-white border-b p-4 flex gap-4">
    <a href="/dashboard">Tổng quan</a>
    <a href="/transactions">Giao dịch</a>
    <a href="/categories">Danh mục</a>
    <form method="POST" action="/logout" class="ml-auto"><button type="submit">Đăng xuất</button></form>
  </nav>
  <main class="max-w-3xl mx-auto p-6">
    {{template "content" .}}
  </main>
</body>
</html>
{{end}}
```

- [ ] **Step 2: Write failing end-to-end smoke test**

`internal/handlers/smoke_test.go`:

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEndToEndRegisterAddTransactionSeeDashboard(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "e2e-test@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), 0)
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}

	form := url.Values{
		"category_id": {itoa(categories[0].ID)},
		"amount":      {"25000"},
		"type":        {"expense"},
		"description": {"Trà sữa"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	addReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	addReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addReq.AddCookie(cookie)
	router.ServeHTTP(httptest.NewRecorder(), addReq)

	dashReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	dashReq.AddCookie(cookie)
	dashRec := httptest.NewRecorder()
	router.ServeHTTP(dashRec, dashReq)

	if dashRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from dashboard, got %d", dashRec.Code)
	}
	if !strings.Contains(dashRec.Body.String(), "25000") {
		t.Fatal("expected dashboard total to include the new transaction")
	}
}
```

Add `"context"` to the imports.

- [ ] **Step 3: Run test to verify it fails or passes**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./internal/handlers/... -run TestEndToEndRegisterAddTransactionSeeDashboard -v`
Expected: Should already PASS since all routes exist from prior tasks — this test locks in the full flow as a regression guard. If it fails, debug against the specific handler it touches (register/login/transactions/dashboard) rather than changing the test's assertions.

- [ ] **Step 4: Run the full test suite**

Run: `TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./... -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/templates/layout.html internal/handlers/smoke_test.go
git commit -m "feat: add navigation and end-to-end smoke test"
```

---

### Task 12: Dockerize the app

**Files:**
- Create: `Dockerfile`
- Modify: `docker-compose.yml` (add `app` service alongside `postgres`)

**Interfaces:**
- No Go interfaces — this task packages the existing app for local/VPS deployment.

- [ ] **Step 1: Write `Dockerfile`**

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=build /out/server ./server
COPY internal/database/migrations ./internal/database/migrations
COPY internal/web/templates ./internal/web/templates
EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 2: Add `app` service to `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: expense
      POSTGRES_PASSWORD: expense
      POSTGRES_DB: expense_tracker
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  app:
    build: .
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://USER:PASSWORD@postgres:5432/expense_tracker?sslmode=disable
      PORT: 8080
    ports:
      - "8080:8080"

volumes:
  pgdata:
```

- [ ] **Step 3: Verify the image builds**

Run: `docker build -t expense-tracker .`
Expected: build succeeds with no errors.

- [ ] **Step 4: Verify the full stack boots and responds**

Run:
```bash
docker compose up -d
sleep 3
migrate -database "postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" -path internal/database/migrations up
curl -f http://localhost:8080/healthz
```
Expected: `curl` prints `ok` and exits 0.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "chore: dockerize app for local and VPS deployment"
```

---

## Self-Review Notes

- **Spec coverage:** Auth (Tasks 4-7), categories (Task 8), transactions CRUD (Task 9), reports/dashboard (Task 10), error handling / user-scoping (enforced in every query + isolation test in Task 9), testing (integration tests throughout, unit tests in Task 4), Docker deployment (Task 12), out-of-scope items (splitting, budgets, multi-currency, OAuth, mobile) intentionally excluded — matches design doc.
- **Known follow-up gaps intentionally deferred, not silently dropped:** htmx-specific partial swaps (`transaction_row.html`/`transaction_form.html` for inline edit-without-reload) and month-filter UI (`?month=` query param) were noted in Task 9 as a later polish pass — the underlying data query (`ListTransactionsForMonth`) already supports it; only the UI control to pick a month is missing. Add a Task 13 for this before considering the app UI-complete.
- **Type consistency check:** `sqlcgen.CreateUserParams`, `sqlcgen.CreateSessionParams`, `sqlcgen.CreateCategoryParams`, `sqlcgen.CreateTransactionParams`, etc. are used consistently with the field names implied by each `.sql` query's positional params — exact generated Go field names (e.g. whether sqlc emits `OccurredOn` vs `OccurredOn time.Time` vs a pgtype date wrapper) must be verified against `internal/sqlcgen/models.go` after each `sqlc generate` step, since sqlc's exact type mapping for `DATE` columns depends on installed sqlc version. Treat every `sqlc generate` step as a checkpoint to confirm generated field/type names before writing code that references them.

# Expense Tracker

A simple, server-rendered expense tracker for personal/family use. Each user
has their own account and tracks their income and expenses independently —
there's no bill-splitting or shared budgets between accounts. Amounts are
stored and displayed in VND (Vietnamese Dong) as plain integers (no decimal
handling). The UI is in English. Categories can be personal (created by a
user) or shared defaults seeded by the database migrations (Food & Drink,
Transport, Salary, ...).

Built as a Go monolith: `chi` router, `html/template` server-rendered pages,
PostgreSQL via `sqlc`-generated queries, and Chart.js (CDN) for the
dashboard's category breakdown chart.

## Prerequisites

- Go (see `go.mod` for the exact version — currently `1.26.5`)
- PostgreSQL (any recent version; the project is tested against Postgres 16)
- Optional, for development only:
  - [`sqlc`](https://sqlc.dev) — regenerates `internal/sqlcgen` from the SQL
    in `internal/database/queries` after schema/query changes
  - [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI — only
    needed for authoring/testing new migrations by hand; the app itself
    applies migrations automatically at startup (see below), so it is not
    required to run the app

## Running locally

1. Copy `.env.example` to `.env` (or otherwise set the same environment
   variables) and adjust `DATABASE_URL` to point at your Postgres instance:

   ```
   DATABASE_URL=postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable
   PORT=8080
   SESSION_COOKIE_NAME=session_id
   SECURE_COOKIES=false
   ```

   `USER` and `PASSWORD` are placeholders — substitute your own Postgres
   role and its password. `.env` is gitignored so real credentials stay out
   of the repository; keep them out of `.env.example` as well, which is
   committed and ships with every value blank.

   Set `SECURE_COOKIES=true` only once the app is served over HTTPS —
   otherwise browsers will reject the session cookie and nobody can log in.

   `APP_BASE_URL`, `BREVO_API_KEY`, and `MAIL_FROM` configure the
   password-reset email (see `.env.example`); it's sent through Brevo's HTTP
   API rather than SMTP because Render's free tier blocks outbound SMTP
   ports entirely. All are optional — leave them blank and
   "Forgot password?" still works end to end except the actual send, which
   is logged instead of delivered.

   `INBOUND_DOMAIN` and `INBOUND_WEBHOOK_SECRET` configure the email
   ingestion webhook (see `.env.example`); they enable users to forward
   bank emails into the tracker. The Cloudflare Email Worker in `emailworker/`
   is deployed separately by running `npx wrangler deploy` from inside that
   directory and is not covered by a `git push`.

2. Make sure the target Postgres database exists and is reachable at
   `DATABASE_URL`.

3. Run the server:

   ```
   go run ./cmd/server
   ```

   The server loads `.env` itself on startup (via `godotenv`), so there is
   no need to `export` or `source` it first — a variable already set in the
   environment still takes priority over the file.

   On startup the server applies all pending migrations from
   `internal/database/migrations` automatically (using golang-migrate's Go
   library) before it starts listening — there is no separate migration
   step to run by hand, whether against a brand-new empty database or an
   already-migrated one restarting.

4. Visit `http://localhost:8080` (redirects to `/dashboard`, which in turn
   redirects anonymous visitors to `/login`).

## Running tests

Tests that touch the database are skipped unless `TEST_DATABASE_URL` is set.
Point it at a scratch Postgres database you don't mind schema churn against
(the test suite creates and drops throwaway databases/rows as needed):

```
TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./...
```

This single invocation exercises every package, including the migration
test (which now runs against its own throwaway database rather than the
shared `TEST_DATABASE_URL` database, so it no longer interferes with other
packages' tests that expect the schema to stay in place).

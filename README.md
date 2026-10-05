# $pend

A simple expense tracker for personal/family use. Each user has their own
account and tracks their income and expenses independently — there's no
bill-splitting or shared budgets between accounts. Amounts are stored and
displayed in VND (Vietnamese Dong) as plain integers (no decimal handling).
The UI is in English. Categories can be personal (created by a user) or
shared defaults seeded by the database migrations (Food & Drink, Transport,
Salary, ...).

## Monorepo layout

Two packages:

- **`server/`** — the backend. A Gin-routed JSON API (`internal/api`)
  under `/api/v1/*`, JWT access tokens + an httpOnly-cookie refresh token, no
  server-rendered HTML; PostgreSQL via `sqlc`-generated queries.
- **`client/`** — the frontend. A Vite + React + TypeScript + Tailwind v4
  single-page app that talks to `server/` entirely over `/api/v1/*`.

See `.claude/context/server.md` and `.claude/context/client.md` for the
full picture of each half.

## Prerequisites

- Go (see `server/go.mod` for the exact version — currently `1.26.5`)
- PostgreSQL (any recent version; the project is tested against Postgres 16)
- Node.js 22+ and [pnpm](https://pnpm.io) (only needed for `client/`)
- Optional, for backend development only:
  - [`sqlc`](https://sqlc.dev) — regenerates `server/internal/sqlcgen` from
    the SQL in `server/internal/database/queries` after schema/query changes
  - [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI — only
    needed for authoring/testing new migrations by hand; the app itself
    applies migrations automatically at startup (see below), so it is not
    required to run the app

## Running the backend (`server/`)

1. Copy `server/.env.example` to `server/.env` (or otherwise set the same
   environment variables) and adjust `DATABASE_URL` to point at your
   Postgres instance:

   ```
   DATABASE_URL=postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable
   PORT=8080
   SESSION_COOKIE_NAME=session_id
   SECURE_COOKIES=false
   ```

   `USER` and `PASSWORD` are placeholders — substitute your own Postgres
   role and its password. `server/.env` is gitignored so real credentials
   stay out of the repository; keep them out of `server/.env.example` as
   well, which is committed and ships with every value blank.

   Set `SECURE_COOKIES=true` only once the app is served over HTTPS —
   otherwise browsers will reject the session cookie and nobody can log in.

   `APP_BASE_URL`, `BREVO_API_KEY`, and `MAIL_FROM` configure the
   password-reset email (see `server/.env.example`); it's sent through
   Brevo's HTTP API rather than SMTP because Render's free tier blocks
   outbound SMTP ports entirely. All are optional — leave them blank and
   "Forgot password?" still works end to end except the actual send, which
   is logged instead of delivered.

2. Make sure the target Postgres database exists and is reachable at
   `DATABASE_URL`.

3. Run the server:

   ```
   cd server
   go run ./cmd/server
   ```

   The server loads `.env` itself on startup (via `godotenv`), so there is
   no need to `export` or `source` it first — a variable already set in the
   environment still takes priority over the file.

   On startup the server applies all pending migrations from
   `server/internal/database/migrations` automatically (using
   golang-migrate's Go library) before it starts listening — there is no
   separate migration step to run by hand, whether against a brand-new
   empty database or an already-migrated one restarting.

4. The server now only answers `/api/v1/*` and `/healthz` — there's nothing to
   visit directly in a browser. Run `client/` (below) and visit *that*
   dev server instead; it proxies `/api/*` to this one.

### Backend tests

Tests that touch the database are skipped unless `TEST_DATABASE_URL` is set.
Point it at a scratch Postgres database you don't mind schema churn against
(the test suite creates and drops throwaway databases/rows as needed):

```
cd server
TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./...
```

This single invocation exercises every package, including the migration
test (which runs against its own throwaway database rather than the shared
`TEST_DATABASE_URL` database, so it doesn't interfere with other packages'
tests that expect the schema to stay in place).

## Running the frontend (`client/`)

```
cd client
pnpm install
pnpm dev
```

Vite's dev server proxies `/api/*` requests to `http://localhost:8080`
(see `client/vite.config.ts`), so run the backend alongside it if you want
real data instead of a blank page. `pnpm build` produces a static
`client/dist/`; `pnpm lint` runs `oxlint`.

## Deployment

- `server/` deploys to Render on every push to `main` (`server/render.yaml`,
  Render's native Go runtime, `rootDir: server`). Postgres is hosted
  separately on Neon — see the comments in `server/render.yaml` for why.
- `client/` deploys to Vercel (`client/vercel.json`), separately from the
  backend.

## License

This project is licensed under the [MIT License](LICENSE).

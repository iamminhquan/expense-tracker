# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A server-rendered expense tracker (Go monolith) for personal/family use. Each
user has their own account and tracks income/expenses independently — no
bill-splitting or shared budgets. Amounts are stored and displayed in VND as
plain integers (no decimals). Categories are either personal (created by a
user) or shared defaults seeded by migrations (Ăn uống, Di chuyển, Lương, ...).

Stack: `chi` router, `html/template` server-rendered pages + htmx for partial
updates, PostgreSQL via `sqlc`-generated queries (`pgx/v5`), Chart.js (CDN)
for the dashboard chart. No JS build step — templates and vanilla JS/htmx
only.

## Commands

Run the server (applies pending migrations automatically on startup, so no
separate migrate step):

```
go run ./cmd/server
```

Requires `DATABASE_URL` etc. — copy `.env.example` to `.env` and adjust. Or
`docker compose up` to run app + Postgres together.

Run the full test suite (DB-touching tests are skipped unless
`TEST_DATABASE_URL` is set — point it at a scratch database, since tests
create/drop throwaway databases and rows):

```
TEST_DATABASE_URL="postgres://expense:expense@localhost:5432/expense_tracker?sslmode=disable" go test ./...
```

Run a single package or test:

```
go test ./internal/handlers/... -run TestEndToEndRegisterAddTransactionSeeDashboard
```

Build:

```
go build ./...
```

Regenerate `internal/sqlcgen` after changing SQL in `internal/database/queries`
or the migrations (requires the `sqlc` CLI):

```
sqlc generate
```

## Architecture

**Request flow**: `cmd/server/main.go` wires everything into a `handlers.Deps`
struct (DB pool, `sqlcgen.Queries`, session manager, parsed templates, cookie
config) and builds the router via `handlers.NewRouter(deps)`. Every handler
takes `deps` as a closure argument rather than a receiver method — see the
`xxxHandler(deps) http.HandlerFunc` pattern throughout `internal/handlers/`.

**Routing** (`internal/handlers/router.go`): `/login`, `/register`, `/logout`
are public. Everything else (`/dashboard`, `/transactions`, `/categories`) is
behind an `auth.RequireAuth` middleware group that reads the session cookie
and injects the user ID into the request context
(`auth.UserIDFromContext`).

**Templates**: Each "page" (`auth`, `categories`, `transactions`,
`dashboard`) is its own `*template.Template` built by parsing `layout.html`
plus that page's own files, registered in `main.go`'s `templates` map keyed
by page name. `internal/handlers/render.go` has two entry points:
- `render(w, r, deps, page, active, data)` — full page, executes the
  `"layout"` block, and (if `active != ""`) injects nav data (`ShowNav`,
  `ActiveNav`, `UserName`, `UserInitial`) by loading the current user.
- `renderNamed(w, r, deps, page, tmplName, active, data)` — renders a named
  sub-template instead of the full layout, for htmx fragment responses (a
  single swapped-in row, a tab body, etc.).

Money/date formatting helpers (`vnd`, `vndSigned`, `vndBalance`, `dateFull`,
`dateShort`) live in `internal/handlers/format.go` and are registered as
template funcs via `handlers.TemplateFuncs()`; the formatting rules (dots as
thousands separators, trailing ₫, dd/mm/yyyy) come from
`docs/design/design_handoff_expense_tracker/SPEC.md`.

**htmx conventions**: Mutation handlers (add/edit/delete transaction or
category) return HTML fragments swapped into the DOM rather than JSON.
Session expiry mid-interaction is handled specially: `auth.RequireAuth`
can't just 3xx-redirect an htmx XHR (it would swap the full login page into
whatever partial element was targeted), so `redirectToLogin` in
`internal/auth/middleware.go` sets the `HX-Redirect` header instead when
`HX-Request: true`, which htmx turns into a real top-level navigation. The
same pattern is used for login/register success in `auth_handlers.go`.

**CSRF** (`internal/csrf/csrf.go`): stateless double-submit-cookie pattern —
no server-side token storage. Every request gets a `csrf_token` cookie;
mutating requests must echo it back either via the `X-CSRF-Token` header
(htmx requests — see the `htmx:configRequest` listener in `layout.html`
that copies the `<meta name="csrf-token">` value into the header) or a
hidden `csrf_token` form field (plain `<form method="POST">` submissions,
e.g. logout).

**Database**: schema lives in `internal/database/migrations/` as numbered
`NNNNNN_description.up.sql`/`.down.sql` pairs applied by golang-migrate.
Hand-written queries live in `internal/database/queries/*.sql`; `sqlc`
generates the Go bindings into `internal/sqlcgen/` (package `sqlcgen`,
`pgx/v5` driver) — never hand-edit files in `internal/sqlcgen/`, edit the
`.sql` and regenerate. Month-based queries (transactions list, dashboard
totals) take explicit `[from, to)` date bounds computed in
`internal/handlers/pg.go`'s `currentMonthRange()`, anchored to
`Asia/Ho_Chi_Minh` (not server UTC) so "this month" lines up with what a
Vietnamese user expects regardless of server timezone.

**Auth**: `internal/auth/session.go` (`Manager`) issues/validates sessions
against the `sessions` table; `internal/auth/password.go` handles
hashing/verification. Sessions last 7 days.

## Commit & PR conventions

- Split commits atomically: one logical change per commit, don't bundle
  multiple unrelated features/fixes into a single commit.
- Commit messages are in English. Subject line must be between 72 and 100
  characters.
- Pull request titles are in English. PR bodies are in Vietnamese and should
  briefly summarize what the PR does.

## Notes

- `docs/design/design_handoff_expense_tracker/SPEC.md` is the source of
  truth for formatting/UI rules referenced from code comments.
- `docs/superpowers/` contains planning docs from prior feature work
  (design specs, implementation plans) — useful background, not living
  documentation.

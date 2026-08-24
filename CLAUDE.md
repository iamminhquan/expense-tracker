# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A server-rendered expense tracker (Go monolith) for personal/family use. Each
user has their own account and tracks income/expenses independently — no
bill-splitting or shared budgets. Amounts are stored and displayed in VND as
plain integers (no decimals). The UI is in English. Categories are either
personal (created by a user) or shared defaults seeded by migrations (Food &
Drink, Transport, Salary, ...); a default carries a `slug` and renders through
`internal/i18n`'s `CategoryName`, while a personal one has a NULL slug and
renders the name its owner typed. Nothing may identify a default category by
its displayed name — match on the slug.

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

Requires `DATABASE_URL` etc. — copy `.env.example` to `.env` and adjust, and
point it at a locally installed Postgres (the project intentionally ships no
Docker/compose setup). `main.go` loads `.env` itself via `godotenv` before
reading config, so no manual `export`/`source` step is needed; a variable
already set in the environment still wins over the file.

Run the full test suite (DB-touching tests are skipped unless
`TEST_DATABASE_URL` is set — point it at a scratch database, since tests
create/drop throwaway databases and rows):

```
TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable" go test ./...
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
template funcs via `handlers.TemplateFuncs()`, alongside `catName`
(`i18n.CategoryName`). The rules are commas for thousands, a trailing ₫, and
a spelled-out month (`11 Aug 2026`); the app was originally specified in the
Vietnamese convention (dots for thousands, `dd/mm/yyyy`), which is why the
helpers exist at all rather than the templates formatting inline.

**The balance** lives in one place: the `header_balance` widget in both nav
bars (`layout.html`). There is no balance card in any page body — that
partial existed once and was deleted. The widget always reports the real
current month, never the month a page happens to be browsing, because it
sits in the layout above the month picker.

It carries forward across months rather than resetting on the 1st: what a
month closes at is exactly what the next one opens with. `MonthlyTotals`
returns that carried-in figure as a third column (`carried_over`) alongside
the month's own two totals, so the one query serves both. Its `WHERE` reaches
back over the user's whole history and each column narrows from there through
its own `FILTER` — read it carefully before changing it, and note the
`::bigint` wrapping the whole subtraction, without which sqlc types it `int32`
and overflows past 2.1 tỷ đồng.

`internal/handlers/balance.go`'s `balanceSummary` struct (built by
`newBalanceSummary`) resolves the percentages in Go rather than the template,
because `html/template` cannot divide and because every percentage here — a
month with no income, a month that overspent its income — is a division that
has to be guarded. Only the balance is cumulative; the ratio bar and its
caption still measure the displayed month against that month's own income.

Both nav bars render the widget and both are in the DOM at once, so every
mutation response returns `header_balance_oob`, which swaps two ids rather
than relying on one selector. Wrapper spans carry `contents` so they leave no
trace in the flex layout.

**Mobile navigation** (`layout.html`'s `nav_mobile_header` and
`mobile_page_header` template blocks): below `md`, the nav collapses into a
two-tier sticky header instead of the desktop `nav_desktop` bar — a slim
top tier (logo + user menu) and a second tier carrying the page title, the
month picker (dashboard/transactions), and an add button (transactions/
categories). `mobile_page_header` is driven entirely by `.ActiveNav` and
whatever `MonthLabel`/`CurrentMonthValue`/`AvailableMonths` the page's own
data already carries, so it takes no page-specific params. Each page renders
it as the first child of its own swappable month/list section rather than
from the layout, so an htmx month switch (which replaces that section)
still carries the header along instead of leaving it behind.

Only category names go through `internal/i18n`. Every other string is written
in English directly in the template or handler that shows it; there is no
message catalog, and a language switcher would be a separate piece of work.

**Theming**: All colour flows through CSS variables declared in
`layout.html`'s `<style>` block and referenced from `tailwind.config` as
`rgb(var(--c-x) / <alpha-value>)`. The variables hold space-separated RGB
channels, not hex — that is what keeps opacity modifiers like `bg-accent/10`
working, so never put a hex value in one. Never hardcode a colour in a
template either (`text-[#6B6862]`, `style="background-color:#FEF7F5"`); add
or reuse a token. The dark palette is declared twice, once under
`@media (prefers-color-scheme: dark) :root:not(.light)` and once under
`:root.dark`, so the three preferences (`auto`/`light`/`dark`) all resolve in
CSS with no load-time JavaScript and no flash. The preference lives in
`users.theme` and is rendered onto `<html class="...">`; `renderNamed`
defaults it to `auto` for pre-auth pages, because `html/template` prints a
missing map key as the literal `<no value>`. Chart.js cannot read CSS
variables, so `dashboard.html` resolves them via `chartColor()` at
construction and rebuilds both charts on the `themechange` event that the
switch (and an OS flip while on `auto`) dispatches.

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
- Branching: small, low-risk changes (a typo, a one-line style tweak, a
  doc-only edit) may be committed straight to `master`. Anything larger or
  riskier — new features, multi-file refactors, behavior changes — goes on
  a `<type>/<short-description>` branch (e.g. `feat/mobile-ux-polish`) and
  gets merged into `master` (`git merge --no-ff`) once done, rather than
  committed directly.
- Commit messages and PR descriptions must not include a "🤖 Generated with
  Claude Code" footer or a Claude Code session link.

## Notes

- The repo carries no design/planning docs. Every rule that still governs
  the code (money format, the 9 shared default categories, the 8-swatch
  palette, the dashboard's comparison lines) is stated in a comment at the
  place it is enforced — keep it that way rather than starting a `docs/`
  tree again.

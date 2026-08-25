# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A server-rendered expense tracker ("$pend", a Go monolith) for personal/family
use. Each user has their own account and tracks income/expenses independently
— no bill-splitting or shared budgets. Amounts are stored and displayed in VND
as plain integers (no decimals). The UI is in English. Categories are either
personal (created by a user) or shared defaults seeded by migrations (Food &
Drink, Transport, Salary, ...); a default carries a `slug` and renders through
`internal/i18n`'s `CategoryName`, while a personal one has a NULL slug and
renders the name its owner typed. Nothing may identify a default category by
its displayed name — match on the slug.

The four pages behind login are `/dashboard` (month totals, comparison lines,
a category doughnut and a 4-month bar chart), `/transactions` (a paged,
filterable month list with inline edit), `/categories`, and `/settings`
(profile/email/password + the theme switch).

Stack: `chi` router, `html/template` server-rendered pages + htmx for partial
updates, PostgreSQL via `sqlc`-generated queries (`pgx/v5`), Tailwind and
Chart.js from CDN. No JS build step — templates and vanilla JS/htmx only.

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

Build, and the checks that must stay clean before any commit:

```
go build ./...
gofmt -l .        # must print nothing
go vet ./...
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

**Routing** (`internal/handlers/router.go`): `/healthz`, `/static/*`,
`/login`, `/register`, `/logout` are public. Everything else (`/dashboard`,
`/transactions`, `/categories`, `/settings`) is behind an `auth.RequireAuth`
middleware group that reads the session cookie and injects the user ID into
the request context (`auth.UserIDFromContext`). `/healthz` exists for the
deploy health check and the keep-alive ping — see `render.yaml`.

**Templates and static assets** (`internal/web`, both `go:embed`ed into the
binary): `web.Templates(funcs)` is the single place the page sets are built
— each "page" (`auth`, `categories`, `transactions`, `dashboard`, `settings`)
is its own `*template.Template` parsing the shared partials plus that page's
own files, so that only one `{{define "content"}}` is ever in scope.
`cmd/server` and the handler tests both call it; do not hand-roll a
`ParseFiles` list.

The shared partials, parsed into every page set: `layout.html` (the shell),
`nav.html` (desktop bar + mobile bottom bar + wordmark), `mobile_header.html`
(the two sticky mobile tiers), `user_menu.html`, `header_balance.html`. Add a
page-specific file to `pageTemplates` in `internal/web/web.go`.

CSS and JS live in `internal/web/static/` and are served at `/static/` by
`web.StaticHandler()` (public route, ETag'd and `Cache-Control: no-cache` —
`embed.FS` has no ModTime to revalidate against). **Never put a `<style>` or
an inline `<script>` back into a template.** `app.css` and `app.js` load from
`<head>`; everything in `app.js` is a delegated listener on `document`,
because `hx-boost` replaces only `<body>` and a head script therefore runs
once per full page load. A page-specific script (`charts.js`,
`categories.js`) must ship as a `<script src>` *inside* that page's swapped
content instead — htmx re-executes it on swap, which is what rebuilds the
dashboard charts on a month switch.

`internal/handlers/render.go` has two entry points:
- `render(w, r, deps, page, active, data)` — full page, executes the
  `"layout"` block, and (if `active != ""`) injects nav data (`ShowNav`,
  `ActiveNav`, `UserName`, `UserInitial`, `Theme`, `HeaderBalance`) by
  loading the current user.
- `renderNamed(w, r, deps, page, tmplName, active, data)` — renders a named
  sub-template instead of the full layout, for htmx fragment responses (a
  single swapped-in row, a tab body, etc.).

`isFragmentRequest` (same file) is what tells a real fragment request
(`HX-Request` alone) from a boosted nav click (`HX-Request` +
`HX-Boosted`); a handler that branches on `HX-Request` alone hands a boosted
click a fragment instead of the page shell.

Money/date formatting helpers (`vnd`, `vndSigned`, `vndBalance`,
`dateShort`, `countOf`, `swatches`) live in `internal/handlers/format.go`
and are registered as template funcs via `handlers.TemplateFuncs()`,
alongside `catName` (`i18n.CategoryName`). The rules are commas for
thousands, a trailing ₫, and a spelled-out month (`11 Aug 2026`); the app was
originally specified in the Vietnamese convention (dots for thousands,
`dd/mm/yyyy`), which is why the helpers exist at all rather than the
templates formatting inline.

**Month, filters, paging** — three small value-object files in
`internal/handlers/`, all built the same way on purpose: parse leniently
from the URL, never error on a malformed value, and offer a
`...FromRequest` variant that reads the *originating* page's URL out of the
`HX-Current-URL` header (a mutation POST/PATCH/DELETE carries no query
string of its own).
- `month.go` — `currentMonthRange`, `monthRangeFor`, `monthRangeFromRequest`,
  `monthLabel`, `pgDate`, and `vietnamLocation`. Every month window is a
  half-open `[from, to)` anchored to `Asia/Ho_Chi_Minh`, not server UTC, so
  "this month" lines up with what a Vietnamese user expects (with a fixed
  UTC+7 fallback if the tzdata isn't in the runtime image).
- `filters.go` — `txnFilters` (search, type, category, min/max amount), the
  0 sentinel that means "not filtering", the nullable sqlc params both the
  list and the count query take, and `transactionsURL`, the canonical
  address pushed via `HX-Push-Url`.
- `paging.go` — `pageSize` (10) and `pager`, which clamps any requested page
  into one that exists.

**The balance** lives in one place: the `header_balance` widget
(`header_balance.html`), rendered by both nav bars. There is no balance card
in any page body — that partial existed once and was deleted. The widget
always reports the real current month, never the month a page happens to be
browsing, because it sits in the layout above the month picker.

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

**The dashboard** (`internal/handlers/report_handlers.go`) builds everything
in Go and hands the templates finished values: `comparisonText` /
`comparisonTextMobile` for the "Last month X · down Y%" lines (two variants,
because the mobile cards share a row and have half the width),
`buildPieData` for the doughnut — top `pieTopN` = 6 categories plus a
synthetic "Other" aggregate so the chart never grows a tail of one-percent
slivers — and `buildBarSeries` for the `barMonths` = 4 month comparison,
zero-padding any month the query returned no row for. Chart data crosses into
JS as `template.JS`-wrapped JSON, which is why category labels are resolved
through `i18n` here rather than by a template func.

**Mobile navigation** (`mobile_header.html`'s `nav_mobile_header` and
`mobile_page_header` blocks): below `md`, the nav collapses into a
two-tier sticky header instead of the desktop `nav_desktop` bar — a slim
top tier (logo + user menu) and a second tier carrying the page title, the
month picker (dashboard/transactions), and an add button (transactions/
categories). `mobile_page_header` is driven entirely by `.ActiveNav` and
whatever `MonthLabel`/`CurrentMonthValue`/`AvailableMonths` the page's own
data already carries, so it takes no page-specific params. Each page renders
it as the first child of its own swappable month/list section rather than
from the layout, so an htmx month switch (which replaces that section)
still carries the header along instead of leaving it behind.

The mobile add-transaction sheet and the desktop quick-add form are both in
the DOM at once, which is why `handleCreateTransaction` reads `ui_source` to
pick which fragment to re-render on a validation failure, and why the
Expense/Income toggle has two endpoints (`category_options` for the desktop
`<select>`, `category_chips` for the sheet's chips).

Only category names go through `internal/i18n`. Every other string is written
in English directly in the template or handler that shows it; there is no
message catalog, and a language switcher would be a separate piece of work.

**Theming**: All colour flows through CSS variables declared in
`static/app.css` and referenced from `static/tailwind-config.js` as
`rgb(var(--c-x) / <alpha-value>)`. The variables hold space-separated RGB
channels, not hex — that is what keeps opacity modifiers like `bg-accent/10`
working, so never put a hex value in one. Never hardcode a colour in a
template either (`text-[#6B6862]`, `style="background-color:#FEF7F5"`); add
or reuse a token — two tests in `templates_layout_test.go` fail on a literal
`rgba(` or `[#hex]` in a template, and on a utility class stranded outside a
`class="..."` attribute. That same file also fails a form control marked
`flex-1` with no width bound, and a bottom-sheet grab handle that has drifted
away from the `[data-sheet-handle]` selector `app.js` looks for.

The dark palette is declared twice, once under
`@media (prefers-color-scheme: dark) :root:not(.light)` and once under
`:root.dark`, so the three preferences (`auto`/`light`/`dark`) all resolve in
CSS with no load-time JavaScript and no flash. The preference lives in
`users.theme` (CHECK-constrained, mirrored by `theme.go`'s `validTheme`) and
is rendered onto `<html class="...">`; `renderNamed` defaults it to `auto`
for pre-auth pages, because `html/template` prints a missing map key as the
literal `<no value>`. Chart.js cannot read CSS variables, so
`static/charts.js` resolves them via `chartColor()` at construction and
rebuilds both charts on the `themechange` event that the switch (and an OS
flip while on `auto`) dispatches.

The category palette is fixed: 8 user-selectable swatches in
`categorySwatches` (`category_handlers.go`) plus the reserved `#A1A1AA` grey
for the "Other" default and the chart's synthetic aggregate. The set is
enforced twice — `isValidSwatch` in Go and `categories_color_check` in
migration 000006.

**htmx conventions**: Mutation handlers (add/edit/delete transaction or
category) return HTML fragments swapped into the DOM rather than JSON, and
each one returns its refreshed out-of-band companions alongside the row —
`header_balance_oob` always, plus `totals_oob` (count, empty state, pager)
on the transactions page. Session expiry mid-interaction is handled
specially: `auth.RequireAuth` can't just 3xx-redirect an htmx XHR (it would
swap the full login page into whatever partial element was targeted), so
`redirectToLogin` in `internal/auth/middleware.go` sets the `HX-Redirect`
header instead when `HX-Request: true`, which htmx turns into a real
top-level navigation. The same pattern is used for login/register success in
`auth_handlers.go`. The settings forms are the exception: they are plain
`hx-boost`ed POSTs that redirect with `?saved=` on success, so a reload or a
back button doesn't re-submit.

**CSRF** (`internal/csrf/csrf.go`): stateless double-submit-cookie pattern —
no server-side token storage. Every request gets a `csrf_token` cookie;
mutating requests must echo it back either via the `X-CSRF-Token` header
(htmx requests — see the `htmx:configRequest` listener in `static/app.js`
that copies the `<meta name="csrf-token">` value into the header) or a
hidden `csrf_token` form field (plain `<form method="POST">` submissions,
e.g. logout).

**Database**: schema lives in `internal/database/migrations/` as numbered
`NNNNNN_description.up.sql`/`.down.sql` pairs applied by golang-migrate.
Hand-written queries live in `internal/database/queries/*.sql`; `sqlc`
generates the Go bindings into `internal/sqlcgen/` (package `sqlcgen`,
`pgx/v5` driver) — never hand-edit files in `internal/sqlcgen/`, edit the
`.sql` and regenerate. Migrations must be data-preserving and idempotent
where they can be: 000006 and 000008 both `UPDATE ... IN PLACE` rather than
delete-and-reinsert, because `transactions.category_id` has no `ON DELETE`
clause and any account with history would break. Month-based queries
(transactions list, dashboard totals) take explicit `[from, to)` date bounds
computed in `internal/handlers/month.go`.

**Auth**: `internal/auth/session.go` (`Manager`) issues/validates sessions
against the `sessions` table; `internal/auth/password.go` handles bcrypt
hashing/verification. Sessions last 7 days. A password change deletes every
*other* session for that user but keeps the current one.

**Deployment** (`render.yaml`): a free Render web service on Render's native
Go runtime (no Dockerfile), with Postgres on Neon rather than Render's own
free tier, which is deleted after 30 days. `autoDeploy` on `main` — a schema
change ships with the same push, since the server migrates at startup. Two
things there are load-bearing: `DATABASE_URL` must be Neon's *direct*
(non-`-pooler`) connection string, because golang-migrate takes a
session-level advisory lock the pooled endpoint doesn't support; and the
binary is built into and started from the repo root, because migrations are
read from a path relative to the working directory even though templates and
static assets are embedded.

## Commit & PR conventions

- Split commits atomically: one logical change per commit, don't bundle
  multiple unrelated features/fixes into a single commit.
- Commit messages are in English. Subject line must be between 72 and 100
  characters.
- Pull request titles are in English. PR bodies are in Vietnamese and should
  briefly summarize what the PR does.
- Branching: small, low-risk changes (a typo, a one-line style tweak, a
  doc-only edit) may be committed straight to `main`. Anything larger or
  riskier — new features, multi-file refactors, behavior changes — goes on
  a `<type>/<short-description>` branch (e.g. `feat/mobile-ux-polish`) and
  gets merged into `main` (`git merge --no-ff`) once done, rather than
  committed directly.
- Commit messages and PR descriptions must not include a "🤖 Generated with
  Claude Code" footer or a Claude Code session link.

## Notes

- The repo carries no design/planning docs. Every rule that still governs
  the code (money format, the 9 shared default categories, the 8-swatch
  palette, the dashboard's comparison lines) is stated in a comment at the
  place it is enforced — keep it that way rather than starting a `docs/`
  tree again.
- `README.md` is written for a human setting the project up. Keep setup
  facts (env vars, Postgres, how tests are run) in sync between the two
  files rather than letting this one drift into a second README.

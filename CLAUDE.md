# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A server-rendered expense tracker ("$pend", a Go monolith) for personal/family use. Each user has their own account and tracks income/expenses independently — no bill-splitting or shared budgets. Amounts are stored and displayed in VND as plain integers (no decimals). The UI is in English. Categories are either personal (created by a user) or shared defaults seeded by migrations (Food & Drink, Transport, Salary, ...); a default carries a `slug` and renders through `internal/i18n`'s `CategoryName`, while a personal one has a NULL slug and renders the name its owner typed. Nothing may identify a default category by its displayed name — match on the slug.

The four pages behind login are `/dashboard` (month totals, comparison lines, a category doughnut and a 4-month bar chart), `/transactions` (a paged, filterable month list with inline edit), `/categories`, and `/settings` (profile/email/password + the theme switch).

Stack: `chi` router, `html/template` server-rendered pages + htmx for partial updates, PostgreSQL via `sqlc`-generated queries (`pgx/v5`), Tailwind and Chart.js from CDN. No JS build step — templates and vanilla JS/htmx only.

## Commands

Run the server (applies pending migrations automatically on startup, so no separate migrate step):

```
go run ./cmd/server
```

Requires `DATABASE_URL` etc. — copy `.env.example` to `.env` and adjust, and point it at a locally installed Postgres (the project intentionally ships no Docker/compose setup). `main.go` loads `.env` itself via `godotenv` before reading config, so no manual `export`/`source` step is needed; a variable already set in the environment still wins over the file.

Run the full test suite (DB-touching tests are skipped unless `TEST_DATABASE_URL` is set — point it at a scratch database, since tests create/drop throwaway databases and rows):

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

Regenerate `internal/sqlcgen` after changing SQL in `internal/database/queries` or the migrations (requires the `sqlc` CLI):

```
sqlc generate
```

## Architecture

**Request flow**: `cmd/server/main.go` wires everything into a `handlers.Deps` struct (DB pool, `sqlcgen.Queries`, session manager, parsed templates, cookie config) and builds the router via `handlers.NewRouter(deps)`. Every handler takes `deps` as a closure argument rather than a receiver method — see the `xxxHandler(deps) http.HandlerFunc` pattern throughout `internal/handlers/`.

**Finding a file in `internal/handlers`**: it is one flat package — Go allows no other shape, since a package cannot span directories and 31 of its files are `_test.go` that must sit beside what they test — so the *filename* carries the grouping instead. Every file starts with the area it belongs to, and a file sorts next to its own tests:

| prefix | what lives there |
| --- | --- |
| `app_` | wiring: `Deps`, the router, the end-to-end smoke test |
| `auth_` | everything pre-login: login/register, password reset, email verification, the lockout |
| `balance_` | the header balance widget and its carry-forward |
| `category_` | the categories page |
| `import_` | the CSV import handler and its mapping form |
| `inbox_` | the Worker webhook that receives forwarded bank email |
| `report_` | the dashboard |
| `req_` | the value objects parsed out of a request: month/scope, filters, paging, route params |
| `settings_` | the settings page: profile, email, password, sessions, theme, account deletion, email tracking |
| `txn_` | the transactions page: list, filters, sort, paging, cross-month, export |
| `view_` | the render seam, the template FuncMap, and the template invariant tests |

Keep the prefix when adding a file; a new area gets a new prefix rather than a bare name. The package's public surface is deliberately three symbols — `Deps`, `NewRouter`, `TemplateFuncs` — so nothing here needs exporting to be reachable from another file, and splitting the areas into real subpackages would mean exporting most of the package to itself.

**Routing** (`internal/handlers/app_router.go`): `/healthz`, `/static/*`, `/login`, `/register`, `/logout` are public. Everything else (`/dashboard`, `/transactions`, `/categories`, `/settings`) is behind an `auth.RequireAuth` middleware group that reads the session cookie and injects the user ID into the request context (`auth.UserIDFromContext`). `/healthz` exists for the deploy health check and the keep-alive ping — see `render.yaml`.

**Templates and static assets** (`internal/web`, both `go:embed`ed into the binary): `web.Templates(funcs)` is the single place the page sets are built — each "page" (`auth`, `categories`, `transactions`, `dashboard`, `settings`) is its own `*template.Template` parsing the shared partials plus that page's own files, so that only one `{{define "content"}}` is ever in scope. `cmd/server` and the handler tests both call it; do not hand-roll a `ParseFiles` list.

The shared partials, parsed into every page set: `layout.html` (the shell), `nav.html` (desktop bar + mobile bottom bar + wordmark), `mobile_header.html` (the two sticky mobile tiers), `month_picker.html` (the one month control, rendered by both breakpoints on both month-scoped pages), `user_menu.html`, `header_balance.html`. Add a page-specific file to `pageTemplates` in `internal/web/web.go`.

CSS and JS live in `internal/web/static/` and are served at `/static/` by `web.StaticHandler()` (public route, ETag'd and `Cache-Control: no-cache` — `embed.FS` has no ModTime to revalidate against). **Never put a `<style>` or an inline `<script>` back into a template.** `app.css` and `app.js` load from `<head>`; everything in `app.js` is a delegated listener on `document`, because `hx-boost` replaces only `<body>` and a head script therefore runs once per full page load. A page-specific script (`charts.js`, `categories.js`) must ship as a `<script src>` *inside* that page's swapped content instead — htmx re-executes it on swap, which is what rebuilds the dashboard charts on a month switch.

`internal/handlers/view_render.go` has two entry points:
- `render(w, r, deps, page, active, data)` — full page, executes the `"layout"` block, and (if `active != ""`) injects nav data (`ShowNav`, `ActiveNav`, `UserName`, `UserInitial`, `Theme`, `HeaderBalance`) by loading the current user.
- `renderNamed(w, r, deps, page, tmplName, active, data)` — renders a named sub-template instead of the full layout, for htmx fragment responses (a single swapped-in row, a tab body, etc.).

`isFragmentRequest` (same file) is what tells a real fragment request (`HX-Request` alone) from a boosted nav click (`HX-Request` + `HX-Boosted`); a handler that branches on `HX-Request` alone hands a boosted click a fragment instead of the page shell.

Money/date formatting helpers (`VND`, `VNDSigned`, `VNDBalance`, `DateShort`, `DateLong`, `Timestamp`, `CountOf`) live in `internal/format`, which takes finished values and returns strings — no request, no `Deps`, no database, which is what lets every money and date rule be tested on its own. `internal/handlers/view_funcs.go` maps them (plus `catName` (`i18n.CategoryName`) and `swatches`) onto the names templates call, via `handlers.TemplateFuncs()`; that mapping stays with the templates. `format.Timestamp` takes a `*time.Location` rather than reaching for the app's own, since nothing else in the package needs a clock. A transaction row does not call either date helper itself: the list wraps its rows in `txnRow`, whose `Date` method picks the format, and the three handlers that answer with a single row call `rowDate` to make the same choice. That indirection exists because `{{range .Transactions}}` hands the row template one row at a time, so a page-level "these rows need their year" flag is not visible from inside it — and because a template reading a key its map lacks prints nothing at all, so a path that skipped the date would ship a silently empty column. The rules are commas for thousands, a trailing ₫, and a spelled-out month (`11 Aug 2026`); the app was originally specified in the Vietnamese convention (dots for thousands, `dd/mm/yyyy`), which is why the helpers exist at all rather than the templates formatting inline.

**Month, filters, paging** — three small value-object files in `internal/handlers/` (the `req_` group), all built the same way on purpose: parse leniently from the URL, never error on a malformed value, and offer a `...FromRequest` variant that reads the *originating* page's URL out of the `HX-Current-URL` header (a mutation POST/PATCH/DELETE carries no query string of its own).
- `req_month.go` — `currentMonthRange`, `monthRangeFor`, `monthRangeFromRequest`, `monthLabel`, `pgDate`, and `vietnamLocation`. Every month window is a half-open `[from, to)` anchored to `Asia/Ho_Chi_Minh`, not server UTC, so "this month" lines up with what a Vietnamese user expects (with a fixed UTC+7 fallback if the tzdata isn't in the runtime image). The same file holds `txnScope`, which is the transactions list's answer to "one month or all of them": `?month=all` resolves to bounds wide enough (`allTimeFrom`/`allTimeTo`) that the month predicate stops narrowing, so the list, the count and the export keep running the one query each already ran. A scope carries the spelling it arrived as, because every link the page builds has to name it and an all-time window formatted as a month reads `0001-01`. **The dashboard is deliberately not a consumer** — its cards and charts are month-against-month and mean nothing over a whole history, so it keeps calling `monthRangeFor`, which has never heard of `all` and treats it as malformed. That is what makes a hand-typed `/dashboard?month=all` land on the current month rather than on something half-rendered, and it is why the picker only offers the entry under `ActiveNav == "transactions"`.
- `req_filters.go` — `txnFilters` (search, type, category, min/max amount), the 0 sentinel that means "not filtering", the nullable sqlc params both the list and the count query take, and `transactionsURL`, the canonical address pushed via `HX-Push-Url`. `Sort` rides in the same value object but is not a filter: it narrows nothing, so `Any`, `ActiveCount` and the badge all leave it out, and `Sorted` is the separate predicate the create handler asks. Its two orders live in `sortOrders`, and the ORDER BY switches on the bound value through a pair of `CASE`s rather than interpolating a column name — an unknown order matches neither and falls back to the `occurred_on DESC, id DESC` the list has always had.
- `req_paging.go` — `pageSize` (10) and `pager`, which clamps any requested page into one that exists.

**CSV import** (`internal/csvimport` + `import_handlers.go`, `import_mapping.go`) reads any CSV with one transaction per row. What a column means is a `Mapping` -- which column plays which role, what order the date parts are in, whether a minus sign marks an expense -- and the format the app exports is just one such Mapping (`ExportMapping`), recognised by `Sniff` so a round trip skips the mapping screen entirely.

`csvimport` never touches the database: the account arrives as a `[]csvimport.Category` and the answer leaves as a `Sheet` or an `Import`, which is what lets every rule about reading a file be tested without Postgres. `Sniff` proposes, `Plan` applies -- and `Plan` no longer judges headers at all, so a mapping that cannot reach a line reports that line rather than refusing the file.

Guessing is by header name first (`headerAliases`, which spells out the toned and untoned Vietnamese spellings rather than carrying a Unicode normaliser), then by content: a column is a date or an amount if `contentShare` of it parses as one, and of whatever columns are left the one that repeats itself most is the category and the one that repeats least is the note. Every guess is rendered into a control the user can change, which is what licenses rules that rough. The exception is the date order: a column whose days never pass the 12th fits both DD/MM and MM/DD, and that is the only wrong guess in the importer that still produces rows that look right, so `Sheet.AmbiguousDate` makes the screen say so -- and a preview whose failures are mostly date failures says the format is probably wrong instead of listing two hundred separate complaints.

`parseAmount` strips currency symbols, spaces and accounting parentheses, and resolves `.` versus `,` by position: with both present the last one is the decimal point; with one present, three digits after it means thousands. "45.000" is therefore forty-five thousand, which is right far more often than it is wrong here. Fractions round to whole đồng and the preview says how many rows were rounded, because refusing them would mean a file in a currency with cents imports nothing at all.

A category name resolves against the defaults through `i18n` and the slug, never against the `name` column, for the same reason the search does. A name that matches nothing is planned as a new personal category -- one per (name, type), matching the table's own uniqueness -- and the preview names them before anything is written. `csvimport.MatchKey` is exported because the handler pairing rows with the categories it just created has to use the same rule; two subtly different ones would leave a row pointing at a category that was never made.

Import is all-or-nothing: one bad line blocks the file. That is not fastidiousness -- a partial import means fixing three lines and re-importing a file whose other 197 are already in, and nothing in the schema can tell that second copy apart. Two identical coffees on one day are a real thing to record, so exact duplicates are counted and reported rather than refused (`countImportDuplicates`, one query over the file's own date range).

All three steps are one handler and no server-side state. The upload form keeps the file in the DOM and every step re-sends it through `hx-include`, so the mapping travels as form fields and nothing half-finished is left behind by a session that walked away. The file is sniffed on every request (that is what column indexes are validated against, and what the mapping screen is re-rendered from), then the upload is rewound and planned. `Import.Fingerprint` is a digest of what was read, echoed back in a hidden field, so a file swapped between steps is refused rather than imported unseen. Row validation deliberately repeats what `handleCreateTransaction` enforces (amount, type, 200-character note, 7-day future limit) -- a laxer second way in would let the importer create rows the form would have rejected.

**The balance** lives in one place: the `header_balance` widget (`header_balance.html`), rendered by both nav bars. There is no balance card in any page body — that partial existed once and was deleted. The widget always reports the real current month, never the month a page happens to be browsing, because it sits in the layout above the month picker.

It carries forward across months rather than resetting on the 1st: what a month closes at is exactly what the next one opens with. `MonthlyTotals` returns that carried-in figure as a third column (`carried_over`) alongside the month's own two totals, so the one query serves both. Its `WHERE` reaches back over the user's whole history and each column narrows from there through its own `FILTER` — read it carefully before changing it, and note the `::bigint` wrapping the whole subtraction, without which sqlc types it `int32` and overflows past 2.1 tỷ đồng.

`internal/handlers/balance_summary.go`'s `balanceSummary` struct (built by `newBalanceSummary`) resolves the percentages in Go rather than the template, because `html/template` cannot divide and because every percentage here — a month with no income, a month that overspent its income — is a division that has to be guarded. Only the balance is cumulative; the ratio bar and its caption still measure the displayed month against that month's own income.

Both nav bars render the widget and both are in the DOM at once, so every mutation response returns `header_balance_oob`, which swaps two ids rather than relying on one selector. Wrapper spans carry `contents` so they leave no trace in the flex layout.

**The dashboard** (`internal/handlers/report_handlers.go`) builds everything in Go and hands the templates finished values: `comparisonText` / `comparisonTextMobile` for the "Last month X · down Y%" lines (two variants, because the mobile cards share a row and have half the width), `buildPieData` for the doughnut — top `pieTopN` = 6 categories plus a synthetic "Other" aggregate so the chart never grows a tail of one-percent slivers — and `buildBarSeries` for the `barMonths` = 4 month comparison, zero-padding any month the query returned no row for. Chart data crosses into JS as `template.JS`-wrapped JSON, which is why category labels are resolved through `i18n` here rather than by a template func.

**Mobile navigation** (`mobile_header.html`'s `nav_mobile_header` and `mobile_page_header` blocks): below `md`, the nav collapses into a two-tier sticky header instead of the desktop `nav_desktop` bar — a slim top tier (logo + user menu) and a second tier carrying the page title, the month picker (dashboard/transactions), and an add button (transactions/categories). `mobile_page_header` is driven entirely by `.ActiveNav` and whatever `MonthLabel`/`CurrentMonthValue`/`AvailableMonths` the page's own data already carries, so it takes no page-specific params. Each page renders it as the first child of its own swappable month/list section rather than from the layout, so an htmx month switch (which replaces that section) still carries the header along instead of leaving it behind.

The mobile add-transaction sheet and the desktop quick-add form are both in the DOM at once, which is why `handleCreateTransaction` reads `ui_source` to pick which fragment to re-render on a validation failure, and why the Expense/Income toggle has two endpoints (`category_options` for the desktop `<select>`, `category_chips` for the sheet's chips).

Only category names go through `internal/i18n`. Every other string is written in English directly in the template or handler that shows it; there is no message catalog, and a language switcher would be a separate piece of work.

**Theming**: All colour flows through CSS variables declared in `static/app.css` and referenced from `static/tailwind-config.js` as `rgb(var(--c-x) / <alpha-value>)`. The variables hold space-separated RGB channels, not hex — that is what keeps opacity modifiers like `bg-accent/10` working, so never put a hex value in one. Never hardcode a colour in a template either (`text-[#6B6862]`, `style="background-color:#FEF7F5"`); add or reuse a token — two tests in `view_layout_test.go` fail on a literal `rgba(` or `[#hex]` in a template, and on a utility class stranded outside a `class="..."` attribute. That same file also fails a form control marked `flex-1` with no width bound, and a bottom-sheet grab handle that has drifted away from the `[data-sheet-handle]` selector `app.js` looks for.

The dark palette is declared twice, once under `@media (prefers-color-scheme: dark) :root:not(.light)` and once under `:root.dark`, so the three preferences (`auto`/`light`/`dark`) all resolve in CSS with no load-time JavaScript and no flash. The preference lives in `users.theme` (CHECK-constrained, mirrored by `settings_theme.go`'s `validTheme`) and is rendered onto `<html class="...">`; `renderNamed` defaults it to `auto` for pre-auth pages, because `html/template` prints a missing map key as the literal `<no value>`. Chart.js cannot read CSS variables, so `static/charts.js` resolves them via `chartColor()` at construction and rebuilds both charts on the `themechange` event that the switch (and an OS flip while on `auto`) dispatches.

The category palette is fixed: 8 user-selectable swatches in `categorySwatches` (`category_handlers.go`) plus the reserved `#A1A1AA` grey for the "Other" default and the chart's synthetic aggregate. The set is enforced twice — `isValidSwatch` in Go and `categories_color_check` in migration 000006.

**htmx conventions**: Mutation handlers (add/edit/delete transaction or category) return HTML fragments swapped into the DOM rather than JSON, and each one returns its refreshed out-of-band companions alongside the row — `header_balance_oob` always, plus `totals_oob` (count, empty state, pager) on the transactions page. Session expiry mid-interaction is handled specially: `auth.RequireAuth` can't just 3xx-redirect an htmx XHR (it would swap the full login page into whatever partial element was targeted), so `redirectToLogin` in `internal/auth/middleware.go` sets the `HX-Redirect` header instead when `HX-Request: true`, which htmx turns into a real top-level navigation. The same pattern is used for login/register success in `auth_handlers.go`. The settings forms are the exception: they are plain `hx-boost`ed POSTs that redirect with `?saved=` on success, so a reload or a back button doesn't re-submit.

**CSRF** (`internal/csrf/csrf.go`): stateless double-submit-cookie pattern — no server-side token storage. Every request gets a `csrf_token` cookie; mutating requests must echo it back either via the `X-CSRF-Token` header (htmx requests — see the `htmx:configRequest` listener in `static/app.js` that copies the `<meta name="csrf-token">` value into the header) or a hidden `csrf_token` form field (plain `<form method="POST">` submissions, e.g. logout).

**Database**: schema lives in `internal/database/migrations/` as numbered `NNNNNN_description.up.sql`/`.down.sql` pairs applied by golang-migrate. Hand-written queries live in `internal/database/queries/*.sql`; `sqlc` generates the Go bindings into `internal/sqlcgen/` (package `sqlcgen`, `pgx/v5` driver) — never hand-edit files in `internal/sqlcgen/`, edit the `.sql` and regenerate. Migrations must be data-preserving and idempotent where they can be: 000006 and 000008 both `UPDATE ... IN PLACE` rather than delete-and-reinsert, because `transactions.category_id` has no `ON DELETE` clause and any account with history would break. Month-based queries (transactions list, dashboard totals) take explicit `[from, to)` date bounds computed in `internal/handlers/req_month.go`.

**Auth**: `internal/auth/session.go` (`Manager`) issues/validates sessions against the `sessions` table; `internal/auth/password.go` handles bcrypt hashing/verification. Sessions last 7 days. A password change deletes every *other* session for that user but keeps the current one.

`internal/auth/lockout.go` holds the login throttle's numbers -- 5 consecutive wrong passwords lock an account for 15 minutes -- because the SQL that stamps the lock and the message that explains it have to agree on them. The state is two columns on `users` (`failed_login_attempts`, `locked_until`) rather than a table of its own, so only real accounts are ever counted and a lapsed lock needs no sweeping. `RecordFailedLogin` counts and locks in one UPDATE, since a read-modify-write in Go would let a parallel flood spend far more than 5 guesses. The lock is checked *before* the password, so guessing at a locked account can't extend the window; a completed password reset clears it, which is the only way out that doesn't involve waiting.

The `/settings` "Active sessions" card is what makes `sessions` rows visible to the account they belong to, rather than something only a password change ever touched. `created_at` and `user_agent` (migration 000012) exist only for this list -- `user_agent` is nullable because a session created before that migration has none. `format.DeviceLabel` (`internal/format/device.go`) turns the raw UA into a name like "Chrome on Windows" by matching a handful of common browser/OS substrings, falling back to the raw string rather than guessing wrong. Signing out one listed device goes through `DeleteSessionForUser`, scoped to `user_id` as well as `id` so a `session_id` typed into the form can never reach a row it doesn't own; "Log out everywhere else" calls the same `DeleteOtherSessionsForUser` a password change uses internally, here as a deliberate action instead of a side effect. The current session never gets its own revoke button, so a click can't log the viewer out of the page they are on.

**Email verification** (`internal/auth/email_verification.go`, `internal/handlers/auth_email_verification.go`) mirrors the forgot-password design against its own `email_verification_tokens` table: a 24h TTL rather than the reset link's 1h, since confirming an address is never the locked-out-owner urgency a password reset is. One token type serves both entry points a link can prove -- a fresh signup and a settings email change -- because both ultimately ask the same question, "can this account be reached here", and `ApplyVerifiedEmail` answers it the same way either time: copy the proven address onto `users.email` and flip `email_verified`.

A settings email change (`updateEmailHandler`) never touches `users.email` directly; it stages the request on `pending_email` and only `ApplyVerifiedEmail` promotes it, once the link sent to the *new* address has been visited. This is deliberate: `users.email` is also the login identity and the address a forgot-password link is sent to, so applying a change immediately would mean a typo can cost the owner both at once, with no way back in. Holding it means a typo just leaves `pending_email` sitting unconfirmed forever -- the owner keeps logging in and recovering their account on the address that was always correct, and can simply resubmit the form. The pre-check that rejects an address already registered excludes the caller's own row, so resubmitting the address already on the account is still a no-op rather than a false collision.

An unverified account is never blocked from anything -- $pend has no bot-signup problem worth locking real people out over -- but every authenticated page carries a small reminder banner (defined in `layout.html`, gated on `EmailVerified` from `authPageData`) until the address is confirmed, with a resend link that reissues whichever address is currently unconfirmed (`pending_email` if a change is in flight, otherwise the account's own `email`). Migration 000013 grandfathers in every account that already existed as verified, since the check postdates their signup and they never asked for it.

**Account deletion** (`deleteAccountHandler` / `deleteAccount` in `settings_handlers.go`) is a hard delete, not a `deleted_at` flag: a grace period would put a "is this account still alive" predicate on every query in the app, and $pend has no billing, audit trail or support desk that a recoverable window would serve. What replaces it sits in the danger zone card instead -- a CSV export link above the delete button, since the history is the part anyone regrets, not the empty account.

`deleteAccount` removes transactions, then the account's own categories, then the user row, in one transaction, spelled out rather than left to the `ON DELETE CASCADE` on `users`. A single cascading delete does work, but only because `transactions.category_id` references `categories(id)` with no `ON DELETE` clause and Postgres defers that NO ACTION check to the end of the statement -- true, invisible to anyone reading the delete, and one constraint change away from not being true. The shared defaults are never at risk: they carry a NULL `user_id`, so `WHERE user_id = $1` cannot reach them. Everything else that points at the account -- sessions, both token tables -- cascades.

The address is released, and re-registering it is a test (`TestDeletedEmailCanRegisterAgainAsAFreshAccount`) rather than an accident. No account shares data with another, so the new signup starts empty; reserving the address would mean keeping a tombstone of the one piece of personal data the owner just asked to be rid of. The gate is the current password, matching the email and password forms, and the confirmation lands on `/login?deleted=1` because by then there is no account left to show it to.

**Email ingestion** (`internal/inbound`, `internal/handlers/inbox_webhook.go`, `emailworker/`): a Cloudflare Email Worker receives mail forwarded to `<token>@in.<domain>`, signs the JSON body with HMAC-SHA256 under `INBOUND_WEBHOOK_SECRET`, and POSTs it to `/inbox/{token}` — a public route, and the one route exempt from CSRF, because the caller is the Worker rather than a browser with a cookie to double-submit. Three layers decide whether to trust it: the token names an account, the signature proves the body came from our own Worker, and the sender domain proves the mail came from a bank; a message failing only the last is stored as `ignored` rather than dropped, so "why did nothing happen" stays answerable. The handler stores the raw message and answers 200 without reading it — parsing lives in a later slice, and a parser that is still wrong must never be able to lose an email. `internal/inbound` holds every rule the Go side shares with the Worker: the payload field names, the HMAC signature, the inbox token, and the body cap. Change one there and `emailworker/src/index.js` changes in the same commit, or email silently stops arriving. The token uses lowercase base32 because mail systems normalise the local part of an address to lowercase; base64url would have been case-sensitive and unmatchable after a real round trip.

**Deployment** (`render.yaml`): a free Render web service on Render's native Go runtime (no Dockerfile), with Postgres on Neon rather than Render's own free tier, which is deleted after 30 days. `autoDeploy` on `main` — a schema change ships with the same push, since the server migrates at startup. Two things there are load-bearing: `DATABASE_URL` must be Neon's *direct* (non-`-pooler`) connection string, because golang-migrate takes a session-level advisory lock the pooled endpoint doesn't support; and the binary is built into and started from the repo root, because migrations are read from a path relative to the working directory even though templates and static assets are embedded. **The repo has a second deploy target that `git push` does not touch**: `emailworker/` is a Cloudflare Email Worker (Email Routing catch-all on `in.<domain>` forwards bank email into it), deployed by running `npx wrangler deploy` from inside that directory. It lives here rather than in its own repo because it shares a signing and payload contract with the inbox handler, but nothing automates it -- change it and forget to deploy, and email silently keeps hitting the old version.

## Go coding conventions

Follow Google's official Go guidance, in its own stated order of priority: **clarity > simplicity > concision > maintainability > consistency**. The sources, in precedence order, are the [Google Go Style Guide](https://google.github.io/styleguide/go/) (Style Guide → Style Decisions → Best Practices), [Effective Go](https://go.dev/doc/effective_go), and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Where this repo has already settled a question, match the surrounding code — a consistent codebase beats a locally-optimal snippet.

**Formatting and tooling.** `gofmt` output is the only accepted formatting; `gofmt -l .` must print nothing and `go vet ./...` must be clean before a commit. There is no line-length limit, but prefer breaking a long expression over letting one line carry three ideas. No linter config is checked in; don't add one without asking.

**Naming.**
- `MixedCaps` / `mixedCaps`, never `snake_case` or `SCREAMING_CASE`. Initialisms keep one case throughout: `userID`, `csrfToken`, `URLParam`, not `userId` or `CsrfToken`.
- Name length scales with scope. A loop or a three-line closure gets `f`, `p`, `m`, `row`; a package-level identifier gets a name that reads on its own. `deps`, `w`, `r`, `ctx` are house names — use them.
- Package names are short, lowercase, single-word, no underscores, no plurals where a singular reads better, and never `util`, `common`, `helpers`, or `base`. The name is part of every reference, so avoid stutter: `csrf.Middleware`, not `csrf.CSRFMiddleware`.
- No `Get` prefix on accessors. `TokenFromRequest`, not `GetToken`. (The sqlc-generated `GetUserByID` is generated from the query name and is exempt.)
- Receivers are one or two letters, consistent across every method on the type: `(m *Manager)`, `(f txnFilters)`, `(p pager)`.

**Errors.**
- Error strings are lowercase and unpunctuated: `"session expired"`, not `"Session expired."`.
- Wrap with `%w` when the caller might reasonably inspect the cause, `%v` when it is just context; always add what was being attempted: `fmt.Errorf("parse %s templates: %w", page, err)`.
- Compare with `errors.Is` and `errors.As`, never `==` or a bare type assertion. The Postgres unique-violation check is the pattern here: `errors.As(err, &pgErr) && pgErr.Code == "23505"`.
- Handle every error. A deliberate discard needs `_ =` and, unless the reason is obvious, a comment. `userID, _ := auth.UserIDFromContext(...)` inside a `RequireAuth` group is the one routine exception — the middleware guarantees the value.
- Sentinel errors are `Err`-prefixed package-level `var`s (`config.ErrMissingDatabaseURL`).
- Don't `panic` in normal flow. The one panic in this repo (`loadStaticAssets`) is a build-time invariant, and says so.

**Control flow.** Keep the happy path at the left margin: handle the error, return early, and leave the success case unindented. Prefer a `switch` to a chain of `else if`. Avoid `else` after a `return`.

**Functions and types.**
- `ctx context.Context` is the first parameter, never a struct field.
- Return concrete types; accept interfaces only where more than one implementation is real. This codebase has almost none by design.
- Named result parameters are for documenting what a multi-value return means (`buildPieData`'s `labels, values, colors, legend`), not for naked returns in long functions — a function you have to scroll to read should return its values explicitly.
- Avoid `any`. `countOf` takes one because two template call sites hand it different integer types; that's the bar.
- Declare an empty slice as `var s []string` unless nil and empty differ to the caller — and when they do, say so, as `searchSlugs` does.
- Prefer a small struct with a constructor over a long parameter list once a value has behaviour of its own (`newPager`, `newBalanceSummary`, `txnFilters`).

**Comments.** Every exported identifier gets a doc comment starting with its own name, in complete sentences. Beyond that, comment the *why*, not the what — a constraint, a workaround, a decision someone would otherwise undo. That is the prevailing style here and the repo depends on it (see Notes below): the reason `header_balance` is out-of-band, the reason `::bigint` wraps that subtraction, the reason month bounds are Vietnam-local. Don't narrate code that already reads clearly.

**Package layout.** Everything lives under `internal/`, one package per responsibility, no import cycles. `internal/web` takes its `template.FuncMap` as an argument specifically so it never imports `internal/handlers`; keep that direction. New shared helpers go in the package that owns the concept, not in a grab-bag. A helper that needs neither a request nor a database belongs outside `internal/handlers` — `internal/format` is where the display strings went, and a new one should either join it or get its own package rather than growing a file in `handlers` that only happens to live there.

**Tests.** Standard library `testing` only — no testify, no assert helpers (`stretchr/testify` in `go.sum` is a transitive dependency of golang-migrate, not ours to use).
- Default to the external test package `package foo_test`, which forces the test through the public API. When a test genuinely needs unexported identifiers, use `package foo` and name the file `*_internal_test.go`.
- Table-driven where the cases are uniform; subtests via `t.Run` when a failure needs a name.
- Failure messages read `FuncName(input) = got, want want`, with `%q` for strings: `t.Errorf("vnd(%d) = %q, want %q", tc.in, got, tc.want)`.
- `t.Fatalf` when the test cannot continue, `t.Errorf` when it can.
- Setup helpers call `t.Helper()` first, and clean up with `t.Cleanup`.
- DB-touching tests read `TEST_DATABASE_URL` and `t.Skip` when it is unset, so `go test ./...` still passes on a machine with no Postgres.
- Tests that assert on template/asset invariants belong in `internal/handlers` next to the existing ones, so the whole suite runs in one invocation.

## Commit & PR conventions

- Split commits atomically: one logical change per commit, don't bundle multiple unrelated features/fixes into a single commit.
- Commit messages are in English. Subject line must be between 72 and 100 characters, counting the prefix.
- Every subject starts with a `<type>: ` prefix, drawn from the same vocabulary the branch names use: `feat`, `fix`, `docs`, `chore`, `style`, `refactor`, `test`. No scope in parentheses. A recent stretch of the history carries no prefix at all; that is drift, not a second convention to match.
- Pull request titles are in English. PR bodies are in Vietnamese and should briefly summarize what the PR does.
- Branching: small, low-risk changes (a typo, a one-line style tweak, a doc-only edit) may be committed straight to `main`. Anything larger or riskier — new features, multi-file refactors, behavior changes — goes on a `<type>/<short-description>` branch (e.g. `feat/mobile-ux-polish`) and gets merged into `main` (`git merge --no-ff`) once done, rather than committed directly.
- Commit messages and PR descriptions must not include a "🤖 Generated with Claude Code" footer or a Claude Code session link.

## Notes

- The repo carries no design/planning docs. Every rule that still governs the code (money format, the 9 shared default categories, the 8-swatch palette, the dashboard's comparison lines) is stated in a comment at the place it is enforced — keep it that way rather than starting a `docs/` tree again.
- `README.md` is written for a human setting the project up. Keep setup facts (env vars, Postgres, how tests are run) in sync between the two files rather than letting this one drift into a second README.

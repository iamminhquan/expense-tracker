# $pend Frontend Context

## Purpose

- This file is the browser-facing source-of-truth for agent work on $pend: everything under `internal/web` (templates, CSS, JS) plus the slice of `internal/handlers` that exists only to feed it (`view_render.go`, `view_funcs.go`, `view_layout_test.go`) and `internal/format` (display strings).
- $pend is a single Go monolith — there is no separate frontend build, no JS framework, no JSON API. A "frontend change" here means a `.html` template, `app.css`, or a file under `internal/web/static/*.js`; it never means a new client project. See `.claude/context/backend.md` for the server/database half of the same monolith — the two files split one codebase by concern, not by deploy target.
- The four pages are `/dashboard`, `/transactions`, `/categories`, `/settings`, plus the pre-auth pages (`/login` /`/register`, `/forgot-password`, `/reset-password`, `/verify-email`). Every one of them is a full `html/template` page on first load and an htmx-swapped fragment on every interaction after.

## Read This First

- Treat this file as the frontend source-of-truth for agent work, alongside `.claude/context/backend.md` (server/DB half) and `.claude/context/README.md` (the directory's own index).
- The detailed rules live in `.claude/rules/*.md`, auto-loaded for the file being touched: `templates-static.md` (the render pipeline and the shared-partial list), `htmx-conventions.md` (fragments, OOB swaps, session-expiry redirects), `mobile-nav.md` (the two-tier mobile header), `theming.md` (CSS variables, the three theme preferences), `balance-widget.md` (the one balance partial both nav bars render), `dashboard.md` (the chart data handoff), `csrf.md` (the header/field a form or htmx request must echo back).
- Prefer reading the actual templates/CSS/JS over this prose — including this file — when they disagree. `internal/web/web.go`'s `pageTemplates` map and `internal/handlers/view_render.go` are the two files that cannot drift without something breaking at parse or execute time, so they are the most reliable sources.

## Maintenance Rule

- Any agent that changes frontend behavior MUST review and update this file (and the relevant `.claude/rules/*.md` file) before finishing work.
- Update this file when any of these change:
  - the page/route surface a template serves (a new page, a renamed fragment)
  - the shared-partial list in `internal/web/web.go` (`sharedTemplates`, `pageTemplates`)
  - the render entry points or `viewData`/`pageView` contract in `internal/handlers/view_render.go`
  - the CSS variable palette or the theme mechanism
  - the CDN dependencies pinned in `layout.html` (htmx, Chart.js, Tailwind Play CDN, fonts)
  - the delegated-listener set in `app.js`, or which script loads from `<head>` vs. inline in a page's own content
- If a change does not affect behavior or layout, no rewrite is required, but check whether the existing text is still accurate.
- Keep updates short and factual — this is not a changelog of template diffs; the Change Log section at the bottom is, one line per change.

## Stack

- Rendering: `html/template`, server-rendered pages, no SPA, no JSON API for the UI. htmx (`unpkg.com/htmx.org@1.9.12`) does every partial update; `hx-boost="true"` on `<body>` turns plain `<a>`/`<form>` navigation into the same fragment-swap machinery without extra markup.
- Styling: Tailwind via the Play CDN (`cdn.tailwindcss.com`), configured at runtime by `static/tailwind-config.js` — no Tailwind build step, no `tailwind.config.js` compiled ahead of time. All color is indirected through CSS variables declared in `static/app.css`.
- Charts: Chart.js (`cdn.jsdelivr.net/npm/chart.js@4.4.4`), driven by `static/charts.js`.
- Fonts: Google Fonts, loaded from `<head>` — Be Vietnam Pro (UI), JetBrains Mono (money/numbers), Playfair Display 800 (the "$pend" wordmark only, subset to five glyphs via `&text=`).
- No JS package manager, no bundler, no transpilation. Every script is plain, pre-ES2017-ish JS (`var`, not `let`/`const` in most of it, to match the existing style) served as-is from `internal/web/static/`, `go:embed`ed into the binary alongside the templates.
- Server glue: `internal/handlers/view_render.go` (the three render entry points), `internal/handlers/view_funcs.go` (the template `FuncMap`), `internal/format` (money/date/count display strings — no request, no DB, testable on its own).

## Main Scope Right Now

- One page shell (`layout.html`) wraps every authenticated page with two nav bars (desktop top bar, mobile bottom bar + two-tier sticky header) and the header balance widget, both of which are always in the DOM and kept in sync by out-of-band swaps.
- Four authenticated pages (`dashboard`, `transactions`, `categories`, `settings`) plus the pre-auth set (`auth` — shared by login/register, `forgot_password`, `reset_password`, `verify_email`).
- Two features carry the most template/JS complexity right now, mirroring the backend's own "most actively evolving" callout: the transactions page (filters, paging, inline edit, mobile action sheets, CSV export/import forms) and the dashboard (the doughnut + bar chart pair, rebuilt in JS on every month switch and every theme change).
- Theming (`auto`/`light`/`dark`) and the mobile long-press/bottom-sheet gesture layer are both load-bearing UI, not polish — see `.claude/rules/theming.md` and `app.js`'s longpress/sheet-drag listeners.

## What Is Actually Implemented

- Page template sets (`internal/web/web.go`'s `pageTemplates`, each parsed on top of `sharedTemplates`):
  - `auth` — `auth.html` + `auth_card_body.html` (login and register share one shell, different card body)
  - `forgot_password`, `reset_password`, `verify_email` — one file each, pre-auth
  - `categories` — `categories.html` + `category_row.html`
  - `transactions` — `transactions.html` + `transaction_form.html` + `transaction_row.html` + `transaction_filters.html`
  - `dashboard` — `dashboard.html`
  - `import` — `import.html` (the CSV mapping form)
  - `settings` — `settings.html` + `settings_inbox.html`
- Shared partials parsed into every page set (`sharedTemplates`): `layout.html`, `nav.html`, `mobile_header.html`, `month_picker.html`, `user_menu.html`, `header_balance.html`.
- Static assets (`internal/web/static/`, served at `/static/` by `web.StaticHandler()`, a public route): `app.css` (design tokens + the handful of rules Tailwind utilities can't express — form-control background, the dot-grid texture, the theme-switch color transition, the `.wordmark` class), `app.js` (every page-independent delegated listener — CSRF header injection, amount-input formatting, long-press sheets, bottom-sheet drag-to-dismiss, button dimming on slow htmx requests, the theme switch, popover dismissal, the transactions export-link rebuild and filter badge, focus restoration after a preserved-node swap, the inbox-token copy button), `tailwind-config.js` (the CDN config, loaded right after the CDN script), `charts.js` (dashboard-only, loaded inline inside the dashboard's content), `categories.js` (categories-only, the mobile add-category sheet's node-moving trick).
- Template func names (`internal/handlers/view_funcs.go`'s `TemplateFuncs()`, mapped onto `internal/format`/`internal/i18n`): `vnd`, `vndSigned`, `vndBalance`, `dateShort`, `catName`, `countOf`, `swatches`.
- Render entry points (`internal/handlers/view_render.go`): `render`, `renderNamed`, `renderFragment` — see Rendering Pipeline below.
- Layout invariant tests (`internal/handlers/view_layout_test.go`): no literal `rgba(`/hex color in a template, no Tailwind utility class stranded outside a `class="..."` attribute, every flexible form control declares a min-width, the bottom-sheet grab handle stays wired to `[data-sheet-handle]`, the transactions export-link rebuild covers every filter control, the transaction row's date cell sizes to whichever date format it gets. Template-set completeness is covered separately in `internal/web/web_test.go` (`TestTemplatesDefinesSharedBlocksInEveryPageSet`, `TestTemplatesDefinesEveryFragmentHandlersRender`).

## Important Reality Checks

- There is no CSS build and no Tailwind config file checked in — `tailwind.config` is a runtime JS object assigned in `static/tailwind-config.js`, read by the Play CDN script on its first DOM scan. Adding a new color means adding a `--c-*` variable to `app.css` **and** a matching Tailwind color key here; one without the other either does nothing or breaks opacity modifiers.
- `app.js` loads from `<head>` and is **never** re-run by an `hx-boost` navigation (boosting only replaces `<body>`), so every listener in it is delegated onto `document` rather than bound to specific elements. A page-specific script (`charts.js`, `categories.js`) is the opposite: it ships as a `<script src>` *inside* the page's own swapped content, specifically so htmx re-executes it on every swap that section receives.
- Color never goes in as a literal. Every color in a template or in `app.css` resolves through a `--c-*` CSS variable holding space-separated RGB channels (never hex — hex silently breaks the `<alpha-value>` opacity modifiers Tailwind composes them with). Two tests in `view_layout_test.go` enforce this on templates; there is no equivalent enforcement inside `app.css` itself beyond review.
- The balance widget (`header_balance.html`) exists in exactly one place in the markup but is rendered twice (once per nav bar) and therefore must be swapped twice — `header_balance_oob` is not optional decoration, it is how both copies stay the same number after a mutation.
- The month picker, the mobile page header, and the balance widget all take **no page-specific parameters** — they read `.ActiveNav` and whatever `MonthLabel`/`CurrentMonthValue`/`AvailableMonths`/`Greeting` the calling page's own data already carries. A new month-scoped page can reuse them as-is only if its data struct already carries those same field names.
- Chart.js cannot read CSS variables — `charts.js` resolves every color it needs via `chartColor()` at construction time, and the whole chart pair is destroyed and rebuilt (not restyled) on the custom `themechange` event, which fires on both an explicit switch click and an OS-level flip while the preference is `auto`.
- The transactions filter form carries `hx-preserve`, which means htmx keeps the DOM node the browser already has across a swap rather than taking the server's freshly rendered copy. That is why the filter badge count and the export-link URL are rebuilt in `app.js` from the live controls instead of being rendered by the server, and why focus/caret position need their own restore logic after a swap — a preserved node survives, but focus on it does not.

## Frontend Layout

- `internal/web/web.go`: owns both `go:embed` trees (`templates`, `static`), builds each page's `*template.Template` set via `Templates(funcs)`, and serves `/static/*` via `StaticHandler()` (ETag'd off one hash of the whole static tree, `Cache-Control: no-cache` since `embed.FS` carries no `Last-Modified`).
- `internal/web/templates/`: one file per page or shared partial — see What Is Actually Implemented above for the grouping. `funcs` (the `template.FuncMap`) is passed in from `internal/handlers`, never imported, so this package stays free of a dependency on it.
- `internal/web/static/`: `app.css`, `app.js`, `tailwind-config.js`, `charts.js`, `categories.js` — see Stack above for what each one owns.
- `internal/handlers/view_render.go`: `render`/`renderNamed`/`renderFragment`, `viewData`/`pageView`, `isFragmentRequest`, `authPageView` (loads the nav/balance/greeting fields every authenticated page needs), `currentHeaderBalance`.
- `internal/handlers/view_funcs.go`: `TemplateFuncs()`, the one place the template-visible function names are declared.
- `internal/format/`: `VND`, `VNDSigned`, `VNDBalance`, `DateShort`, `DateLong`, `Timestamp`, `CountOf`, `GreetingLine` — pure string formatting, no request, no `Deps`, no database, which is what lets every money/date rule be unit-tested in isolation.
- `internal/i18n/`: `CategoryName`, the slug→display-name mapping the `catName` template func calls through — the one piece of "frontend" text that is not simply written in the template, because a default category's name must never be matched or hardcoded, only its `slug`.
- Every other `internal/handlers/*.go` file builds the data a template renders (`balance_`, `category_`, `report_`, `txn_`, `settings_`, …) but is covered as backend logic in `.claude/context/backend.md`; this file's concern is what happens once that data reaches a template.

## Rendering Pipeline

- Three entry points in `internal/handlers/view_render.go`, in increasing order of "how much of the shell does this response carry":
  - `render(w, r, deps, page, active, data)` — the full page: executes `"layout"`, and (when `active != ""`) loads nav data (`ShowNav`, `ActiveNav`, `UserName`, `UserInitial`, `Greeting`, `Theme`, `HeaderBalance`, `EmailVerified`) by reading the current user. Pre-auth pages pass `active=""` so `viewData`'s zero value leaves `layout.html`'s `{{if .ShowNav}}` blocks closed.
  - `renderNamed(w, r, deps, page, tmplName, active, data)` — renders a named sub-template instead of `"layout"`, for an htmx fragment response that still needs the shell's shared fields (a month section, a settings tab body). Still loads nav data when `active != ""`.
  - `renderFragment(w, r, deps, page, tmplName, data)` — renders a named sub-template against `data` exactly as given, adding nothing. For fragments whose templates read nothing from the shell: a swapped-in row, an inline edit form, a confirm prompt.
- **Every call takes a struct, never a `map[string]any`.** A page's data type embeds `viewData` and satisfies `pageView` by being passed as a pointer; `html/template` promotes the embedded struct's fields, so `{{.CSRFToken}}` reaches it from any page without the template knowing it is nested. This was `map[string]any` once — a template reading a missing key renders blank with no error, so a forgotten field shipped a silently blank row rather than a build failure. Keep this contract when adding a page.
- `isFragmentRequest(r)` distinguishes an explicit htmx fragment request (`HX-Request: true` alone, e.g. the month dropdown's `hx-get`) from a boosted top-level nav click (`HX-Request: true` **and** `HX-Boosted: true`) — a handler that branches on `HX-Request` alone would hand a boosted nav click a bare fragment instead of the full page shell.

## Template and Static Asset Conventions

- `web.Templates(funcs)` is the single place page sets are built; `cmd/server` and the handler tests both call it — never hand-roll a `ParseFiles` list elsewhere. Each page gets its own `*template.Template` (not one set holding every file) because several pages define a block of the same name (`"content"` above all), and a single shared set would let the last one parsed win silently.
- Adding a page-specific template file means adding it to `pageTemplates` in `internal/web/web.go`; adding a new shared partial (rare) means adding it to `sharedTemplates` there instead, which puts it in every page's set whether that page uses it or not.
- **Never put a `<style>` or an inline `<script>` into a template.** `app.css`/`app.js` load from `<head>`. A page-specific script ships as a `<script src>` *inside* that page's own swapped content (see Important Reality Checks above for why).
- `internal/handlers/view_layout_test.go` enforces several of these conventions mechanically (no literal color, no stranded Tailwind class, flexible form controls declare a min-width, the sheet-handle selector stays matched) — a template change that trips one of these tests is the convention catching a real regression, not a false positive to work around.

## Theming

- All color flows through CSS variables declared in `static/app.css` and referenced from `static/tailwind-config.js` as `rgb(var(--c-x) / <alpha-value>)`. The variables hold space-separated RGB channels, not hex, which is what keeps opacity modifiers like `bg-accent/10` working — never put a hex value in one.
- The dark palette is declared twice: once under `@media (prefers-color-scheme: dark) :root:not(.light)` and once under `:root.dark`, so all three preferences (`auto`/`light`/`dark`) resolve in CSS alone, with no load-time JavaScript and no flash of the wrong palette.
- The preference lives in `users.theme` (DB-backed, CHECK-constrained, mirrored by `settings_theme.go`'s `validTheme`) and is rendered onto `<html class="...">`. `renderNamed` defaults it to `auto` for pre-auth pages, which have no user to read a preference from.
- The switch itself (`app.js`) sets `document.documentElement.className` immediately (no round trip needed, since the `hx-put` response carries no markup) and dispatches a `themechange` custom event, which is what tells Chart.js to rebuild — Chart.js reads colors once at construction and cannot react to a CSS variable changing on its own.
- The category palette is fixed: 8 user-selectable swatches (`categorySwatches` in `category_handlers.go`, exposed to templates via the `swatches` func) plus the reserved `#A1A1AA` grey for the "Other" default and the chart's synthetic aggregate slice. Enforced twice — `isValidSwatch` in Go and a CHECK constraint in migration 000006.

## htmx Conventions

- Mutation handlers (add/edit/delete transaction or category) answer with HTML fragments swapped into the DOM, never JSON. Every one returns `header_balance_oob` as an out-of-band swap alongside its main response — both nav bars render that widget and have to stay in sync — and the transactions page additionally returns `totals_oob` (count, empty state, pager).
- Session expiry mid-interaction can't be a plain redirect: `auth.RequireAuth` 3xx-ing an htmx XHR would swap the full login page into whatever partial element the request targeted. `redirectToLogin` in `internal/auth/middleware.go` sets `HX-Redirect` instead when `HX-Request: true`, which htmx turns into a real top-level navigation. Login/register success in `auth_handlers.go` uses the same pattern.
- The settings forms are the deliberate exception to "fragments, not full responses": they are plain `hx-boost`ed POSTs that redirect with `?saved=` on success, so a reload or a back-button press doesn't resubmit the form.
- `hx-boost="true"` on `<body>` is what turns an ordinary `<a>`/`<form>` into the same swap machinery, which is also why a `<head>` script runs exactly once per full page load (boosting replaces only `<body>`) — see Important Reality Checks above.

## CSRF (Client Side)

- Stateless double-submit-cookie pattern; the server half lives in `internal/csrf/csrf.go`, covered in `.claude/context/backend.md` and `.claude/rules/csrf.md`. The frontend's half is two lines: a `<meta name="csrf-token">` tag in `layout.html`'s `<head>`, and the `htmx:configRequest` listener in `app.js` that copies its value into the `X-CSRF-Token` header on every htmx request.
- A plain `<form method="POST">` that does not go through htmx (logout, the email-verify-banner's resend button) instead carries a hidden `csrf_token` field set from `{{.CSRFToken}}` — both paths read from the same `viewData.CSRFToken` the render pipeline fills in.

## The Dashboard and Charts

- `internal/handlers/report_handlers.go` builds every value the templates/JS need in Go and hands them over finished — `comparisonText`/`comparisonTextMobile`, `buildPieData` (top 6 categories + a synthetic "Other" aggregate so the doughnut never grows a tail of one-percent slivers), `buildBarSeries` (4-month comparison, zero-padded for a month with no rows). Category labels are resolved through `i18n` in Go here, not through a template func, because chart data crosses into JS as `template.JS`-wrapped JSON rather than through `html/template`'s normal escaping.
- `charts.js` owns `window.__initCharts()`, which (re)builds both the doughnut and the bar chart from a `#chart-data` JSON blob in the page. It is loaded by a `<script src>` at the end of the dashboard's month section, not from `<head>`, specifically so an htmx month switch (which swaps that whole section) re-executes it and rebuilds the charts against the new data.
- The doughnut's tooltip is a hand-built DOM node (`pieTooltipEl`), not Chart.js's own canvas-painted tooltip — on a 108px mobile doughnut the built-in tooltip clips at the canvas edge, so an external tooltip free to overflow and clamp to the viewport is used instead.
- Every rebuild tears down the previous `window.__pieChart`/`window.__barChart` and detaches the previous `themechange`/`pointerdown`/`scroll`/`resize` listeners before re-attaching — `charts.js` re-runs on every dashboard render, and skipping this would leave one more copy of each listener behind per month switch.

## Mobile Navigation

- Below `md`, the desktop nav bar (`nav_desktop`) is hidden and replaced by a two-tier sticky header: `nav_mobile_header` (logo + balance + user menu) and `mobile_page_header` (page title + month picker + add button), both in `mobile_header.html`. Each page renders `mobile_page_header` as the first child of its own swappable month/list section — not from the layout — so an htmx month switch that replaces that section carries the header along instead of leaving it behind.
- The mobile add-transaction sheet and the desktop quick-add form are both in the DOM at once. `handleCreateTransaction` reads `ui_source` to pick which fragment to re-render on a validation failure, and the Expense/Income toggle has two endpoints for the same reason: `category_options` feeds the desktop `<select>`, `category_chips` feeds the sheet's chips.
- The categories page's single add-category form *moves* between a desktop sidebar slot and a mobile bottom sheet rather than being rendered twice (`categories.js`'s `openAddCategorySheet`) — moving the node is what keeps a validation error, a typed name, and the `hx-post` target in one place regardless of which breakpoint the form is currently showing on.
- Long-press (~500ms, via `[data-longpress-target]`) opens a mobile transaction row's action sheet; bottom sheets themselves are dismissed by dragging the grab handle down past a quarter of their height, or a short flick (`app.js`'s pointer-event listeners — unified touch/mouse, so a scroll that steals the gesture arrives as `pointercancel` and cancels the press for free).
- Only category names go through `internal/i18n`. Every other string is written in English directly in the template or handler that shows it — there is no message catalog, and a language switcher would be new infrastructure, not a flag to flip.

## Files Agents Should Prefer Reading Before Edits

- `internal/web/web.go` — the actual `pageTemplates`/`sharedTemplates` lists; the real page-to-file mapping.
- `internal/handlers/view_render.go` — the render entry points and the `viewData`/`pageView` contract.
- `internal/handlers/view_funcs.go` — the exact set of template-callable function names.
- `internal/web/static/app.css` — the full `--c-*` variable set, both palettes.
- `internal/web/static/app.js` — every page-independent delegated listener; read it before adding a new global behavior to make sure one doesn't already exist.
- `internal/web/templates/layout.html` — the CDN script/stylesheet order (it matters: `app.css` before Tailwind, `tailwind-config.js` after the Tailwind CDN script).
- `internal/handlers/view_layout_test.go` and `internal/web/web_test.go` — the invariants a template/asset change must not break.
- `.claude/rules/*.md` — area-specific rules, auto-loaded for the file being touched.

## Safe Edit Rules For Agents

- Never put a literal hex color or `rgba(...)` in a template or a hand-written style attribute — add or reuse a `--c-*` token in `app.css`. `view_layout_test.go` enforces this on templates mechanically; it does not check `app.css` itself, so review a new variable by eye.
- Never write a `<style>` block or an inline `<script>` into a template. `app.css`/`app.js` are the only places those belong, and a page-specific script goes inside that page's own swapped content, not `<head>`.
- Preserve the render-entry-point contract in `view_render.go` (struct, never a map) when adding a new page or fragment.
- A mutation handler that changes anything the header balance depends on must return `header_balance_oob` alongside its main response — both nav bars render the widget and nothing else keeps them in sync.
- Keep `app.css`'s variable list and `tailwind-config.js`'s color map in lockstep — a variable added to one without the other either does nothing (Tailwind) or breaks opacity modifiers (CSS).
- A page-independent behavior belongs as a delegated listener in `app.js`, bound to `document` — not to a specific element, since `hx-boost` only swaps `<body>` and a listener bound to an element the first page happened to render will not survive a boosted navigation to a different page.
- Follow `.claude/rules/go-conventions.md` for the Go half of this layer (`view_render.go`, `view_funcs.go`, `internal/format`); there is no separate style guide for the templates/CSS/JS, so match the voice and structure of the surrounding file (see the comment density in `app.js`/`charts.js`/`app.css` — every non-obvious choice gets a short "why", not a restatement of the code).

## Common Task Playbooks

- Add a new authenticated page: add the route in `internal/handlers/app_router.go`, add the page's template set to `pageTemplates` in `internal/web/web.go`, write a data struct embedding `viewData` and satisfying `pageView`, render it with `render`/`renderNamed`/`renderFragment` as appropriate (see Rendering Pipeline above), and reuse `month_picker`/`mobile_page_header`/`header_balance` as-is only if the page's data already carries the field names they read.
- Add a new CSS color token: add the `--c-*` variable to both palette blocks in `app.css` (the `@media (prefers-color-scheme: dark) :root:not(.light)` block and the `:root.dark` block), then add the matching key to `tailwind-config.js`'s `theme.extend.colors` as `rgb(var(--c-x) / <alpha-value>)`.
- Add a page-specific script: ship it as a `<script src>` at the end of that page's swapped content (see `charts.js`/`categories.js` for the pattern), not from `<head>` — and if it binds to `document` rather than reacting to its own re-execution, make sure it tears down any listener from a previous run first, the way `charts.js` does.
- Add a new page-independent JS behavior: add a delegated listener to `app.js`, bound to `document`, never to a specific element.
- Change a mutation handler's response: check whether anything it changed affects the header balance (return `header_balance_oob`) or the transactions totals/pager (return `totals_oob`), and check `.claude/rules/htmx-conventions.md` for the session-expiry/`HX-Redirect` pattern if the handler sits behind auth.

## Known Gaps and Debt

- No CSS build step and no compiled Tailwind config — the Play CDN recompiles the whole utility set from scratch on every page load via a client-side scan, which is the accepted trade-off for "no JS build step, no bundler" rather than an oversight.
- No automated check that `app.css`'s variables and `tailwind-config.js`'s color map stay in lockstep beyond review; a variable added to one without the other fails silently (a missing/no-op utility class) rather than loudly.
- There is no message catalog / i18n layer beyond default-category names — every other string is English, hardcoded in templates and handlers. A real language switcher is unbuilt infrastructure, not a flag.
- `view_layout_test.go`'s invariants cover templates only; `app.css`, `app.js`, and the other static files have no equivalent automated check and rely on review.

## Change Log

- `2026-10-04`: `frontend.md` added to `.claude/context/` as the browser-facing counterpart to `backend.md` — templates, static assets, the render pipeline, theming, htmx conventions, and the dashboard/mobile-nav JS.
- `2026-10-04`: `overview.md`, `stack.md`, and `request-routing.md` removed from `.claude/context/` — superseded by `backend.md`/this file and `README.md`'s own directory index. The "Read This First" cross-references here were updated accordingly.

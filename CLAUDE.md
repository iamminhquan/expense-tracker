# $pend

Monorepo: `server/` (Go backend) and `client/` (React + Vite frontend). This
is a migration in progress — see "Migration in Progress" below before
assuming either half's stack from its directory name alone.

The project knowledge lives under `.claude/`; read only what the task needs.

@.claude/context/README.md

Then, by task:

- Backend work (routes, data model, auth, deploy, how to run, test and build) -> `.claude/context/backend.md`
- Frontend work (templates, static assets, rendering) -> `.claude/context/frontend.md`
- Coding rules for an area -> `.claude/rules/`, loaded on their own when you touch a matching file
- Committing or opening a pull request -> `.claude/skills/`

## Migration in Progress

$pend is being converted from a single Go monolith (Chi router,
server-rendered `html/template` + htmx) into this two-package monorepo: a
Gin JSON API in `server/` and a React SPA in `client/`.

- **`server/`** holds two route trees in one process now, dispatched by
  path prefix in `cmd/server/main.go` via `http.ServeMux`:
  - The original Chi + `html/template` app, relocated as-is (see `git mv`
    history on this directory), still serving every page a user actually
    sees. `.claude/context/backend.md`, `.claude/context/frontend.md`, and
    `.claude/rules/*.md` describe this half, with every path written
    relative to `server/` (e.g. `server/internal/handlers/`) to match.
  - A new JSON API under `/api/*` (package `internal/api`, Gin), additive
    and not yet linked from anywhere a real user reaches. **Phase 1 (the
    JSON API rewrite) is complete -- every route the HTML side has now has
    a JSON equivalent**: JWT access tokens (`internal/auth/jwt.go`, stateless, 15 min TTL) plus
    a refresh token that's `internal/auth/session.go`'s existing token
    reused as-is (httpOnly cookie, `Path=/api`) rather than a second
    revocable-token mechanism (`jwt.go`'s doc comment has the reasoning);
    full auth (`/api/register|login|refresh|logout|me`); categories
    (`/api/categories`, one `PATCH` where the HTML side splits
    `.../color`/`.../name`); transactions (`/api/transactions`, same
    filter/month-scope/paging semantics via a duplicated
    `transaction_query.go` — see `.claude/rules/req-value-objects.md`);
    dashboard (`/api/dashboard`, same pie/bar aggregation but raw numbers
    instead of pre-formatted strings — see `.claude/rules/dashboard.md`);
    and settings (`/api/settings` + profile/email/password/theme/
    sessions/account-deletion, same rules as `settings_handlers.go` — see
    `.claude/rules/account-deletion.md` and `.claude/rules/auth-sessions.md`);
    forgot/reset-password + email verification
    (`/api/forgot-password`, `/api/reset-password`, `/api/verify-email` —
    see `.claude/rules/email-verification.md`); and CSV import/export
    (`/api/transactions/import`, `/api/transactions/export` — the HTML
    side's three-screen upload/mapping/preview flow collapsed into one
    endpoint driven by what the client sends, see
    `.claude/rules/csv-import.md`). Every endpoint above is DB-tested end
    to end (`internal/api/*_test.go`, needs `TEST_DATABASE_URL`). See
    `.claude/rules/json-api-conventions.md` for conventions spanning this
    whole package (never ship a `null` where the client expects `[]`,
    refresh-cookie `SameSite` must track `SecureCookies`, duplication over
    sharing with `internal/handlers`) -- both the array/`null` and the
    cookie rule exist because a real browser smoke test caught each one
    crashing/breaking something `go test` had no way to notice; see that
    file before assuming Go's test suite passing means a JSON/cookie
    change actually works. No dedicated context file for `internal/api`
    yet — one gets written once Phase 2 is underway and the route surface
    has stopped shifting day to day.
- **`client/`** is a Vite + React + TypeScript + Tailwind v4 SPA (pnpm
  workspace), **Phase 2 underway**: routing, auth (login/register/
  forgot-password/reset-password/verify-email, access token in memory via
  `AuthContext`, silent refresh on load), all four main pages (Dashboard
  with Chart.js, Transactions with filters/paging, Categories, Settings),
  and CSV import/export (`src/pages/ImportPage.tsx`, an export button on
  Transactions) have a working first pass, each talking to its
  `internal/api` counterpart through `src/hooks/` (TanStack Query) and
  `src/lib/api/`. Verified end to end in a real browser repeatedly
  (register → dashboard → add a category/transaction → settings → dark
  mode → mobile viewport; separately, a full CSV import round trip), which
  is what caught every bug `.claude/rules/json-api-conventions.md`
  documents, including a third one found this way: `TransactionsPage` kept
  its filters in local `useState` instead of the URL, so a CSV import's
  "view transactions" link to a specific month landed back on the current
  month instead -- fixed by making the URL's query string the source of
  truth (`useSearchParams`, not `useState`) the way the HTML side's
  canonical-URL links always worked. Mobile gesture parity landed too --
  `src/hooks/useLongPress.ts` (a ~500ms hold, cancelled by a 10px move the
  same way a scroll cancels it) and `src/components/BottomSheet.tsx` (a
  native `<dialog>`, drag-the-handle-down-to-dismiss past a quarter of its
  height or a short flick) port `server/internal/web/static/app.js`'s two
  pointer-event IIFEs; `TransactionRow` wires a long-press to open the same
  Edit/Delete choice its always-visible text buttons offer, an added
  affordance rather than a replacement. Verified in a real mobile-viewport
  browser session (long-press opens the sheet, a drag dismisses it).
  Every page is its own lazy-loaded chunk (`App.tsx`'s `lazy()` imports,
  one `<Suspense>` boundary around the whole route tree) -- Chart.js alone
  is ~170KB, and splitting by route means a signed-out visitor's first
  paint never fetches it. Known simplifications versus the HTML side, not
  yet addressed: styling is a faithful-effort port of the design tokens
  rather than a pixel-exact match of every template. Dashboard's month
  picker was brought in line with Transactions' URL-as-source-of-truth fix
  too. No dedicated context file yet — one gets written once the page set
  stabilizes.
- Deploy target: `server/` on Render (`server/render.yaml`, `rootDir:
  server`), `client/` on Vercel (`client/vercel.json`) once cutover happens.
  Today only `server/` is deployed; the old single-service Chi app is still
  what's live in production.

Update this section (and the Change Log in `.claude/context/README.md`) as
each migration phase actually lands — don't let it go stale.

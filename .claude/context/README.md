# $pend Project Context

## Purpose
- This directory is the centralized project knowledge base for AI agents working in this repository.
- Canonical architecture and background context lives here. Area-specific coding rules live in `.claude/rules/` (loaded automatically for the file being touched, via `paths:` frontmatter), and step-by-step processes live in `.claude/skills/`.
- `CLAUDE.md` at the repo root is intentionally minimal: it only routes to this directory, `.claude/rules/` and `.claude/skills/`. The common commands live in `backend.md`.

## Context Files
This directory holds exactly two subject-matter files, both comprehensive single documents rather than a pile of short topic pages:
- `backend.md`: the backend source-of-truth — product description, tech stack, the full route surface, package layout, the data model, auth/authorization, env vars, deploy notes, known gaps, and agent playbooks/safe-edit rules. It is the file to update first when backend behavior changes.
- `frontend.md`: the browser-facing source-of-truth — the React page/route structure, the API client layer, auth/theme contexts, TanStack Query's cache strategy, the dashboard/charts, and mobile navigation. It is the file to update first when frontend behavior changes.

Both overlap the `.claude/rules/*.md` files by design (same facts, read in one place here instead of jumping file to file), and both cross-reference each other and this README. Where a context file and a rule disagree, the rule is the source: fix the context file to match it, and fix both to match the code.

## Routing Rule
- For backend questions (routes, schema, auth, env vars, deploy, known gaps, task playbooks), read `backend.md`.
- For frontend questions (pages, components, the API client, theming, charts, mobile nav), read `frontend.md`.
- For coding conventions and rules scoped to a specific area (templates, CSV import, the balance widget, auth/sessions, CSRF, the database, email ingestion, deployment, etc.), don't read this directory — `.claude/rules/*.md` load automatically based on the file being touched, via their own `paths:` frontmatter.
- For committing changes or opening a pull request, read `.claude/skills/`.

## Maintenance Rule
- Any agent changing behavior or operational assumptions in this codebase MUST update the relevant file in `.claude/context/` or `.claude/rules/` before finishing.
- Update context when changing:
  - routes or the `server/internal/api` package layout
  - auth or token behavior (access/refresh tokens, CORS)
  - the database schema or migrations
  - environment variables
  - deployment flow (Render for `server/`, Vercel for `client/`, Neon for Postgres)
  - architectural boundaries between packages
  - important conventions or source-of-truth files

## Repository Map
A monorepo with two packages, `server/` (Go/Gin JSON API) and `client/`
(React/Vite SPA) — see `backend.md` and `frontend.md` for what each one
actually does:
- `server/cmd/server/`: the entrypoint; wires `api.Deps`, runs pending migrations, starts the server.
- `server/internal/api/`: every HTTP handler — one flat package, grouped by filename prefix (see `backend.md`'s Backend Layout section).
- `server/internal/database/`: migrations and hand-written SQL queries; `server/internal/sqlcgen/` holds the generated bindings (never hand-edited — edit the `.sql` and regenerate).
- `server/internal/auth/`: JWT access tokens, the opaque refresh-token/session row, passwords, lockout, password reset, email verification.
- `server/internal/csvimport/`: CSV import/export sniffing, mapping, planning (no database access).
- `server/internal/format/`, `server/internal/i18n/`, `server/internal/txnrule/`, `server/internal/pgval/`: shared helpers (the one surviving display formatter — `DeviceLabel` — category names, transaction limits, pgtype wrappers).
- `server/internal/mailer/`: Brevo HTTP API client for transactional email.
- `client/src/`: the React SPA (Vite + TypeScript + Tailwind v4 + pnpm workspace) — pages, components, hooks, and the `lib/api/` client. See `frontend.md`.
- `.claude/rules/`: path-scoped coding rules, one file per area, each with its own `paths:` frontmatter covering `server/` and/or `client/` files.
- `.claude/skills/`: step-by-step processes (committing changes, opening a pull request).

## Documentation Trust Rule
- Do not assume the root `README.md` is fully current.
- Prefer code and the files in `.claude/context/` and `.claude/rules/` when documentation conflicts.

## Change Log
- `2026-10-05`: the Chi/html-template era ended. `server/internal/handlers`, `server/internal/web`, `server/internal/csrf`, and the unused `server/internal/format` helpers that only templates called (`count.go`, `date.go`, `greeting.go`, `money.go`) were deleted, and `go-chi/chi/v5` dropped from `go.mod`. `cmd/server/main.go` now runs a single Gin router (`internal/api`) — the dual-router dispatch-by-prefix setup described in earlier entries below no longer exists. `CLAUDE.md`, this file, `backend.md`, and `frontend.md` rewritten to describe the final architecture directly, with no "Migration in Progress" framing left anywhere. `.claude/rules/csrf.md`, `htmx-conventions.md`, and `templates-static.md` deleted (no CSRF, no templates, no static-asset embedding left to document).
- `2026-10-05`: `client/` gained a CSV import page (`ImportPage.tsx`, driving `/api/transactions/import`'s mapping/preview/confirm flow) and an export button on Transactions (`downloadTransactionsExport`, fetch+blob since a plain link can't carry the `Authorization` header). A browser smoke test of the full import round trip caught two more bugs: `csvimport.DateFormat`/`NewCategory`/`RowError` have no JSON tags of their own, so embedding them directly in a response leaked PascalCase field names (`dateFormatDTO`/`importNewCategoryDTO`/`rowErrorDTO` now wrap them — see `.claude/rules/json-api-conventions.md`'s new third bullet); and `TransactionsPage` kept filters in local `useState` instead of the URL, so navigating to `/transactions?month=2026-02` (the import flow's own "view results" link) silently showed the current month instead — fixed by switching to `useSearchParams` as the source of truth.
- `2026-10-05`: `internal/api` gained CSV import (`POST /api/transactions/import`, the HTML side's three-screen upload/mapping/preview flow collapsed into one endpoint driven by what the client sends) and export (`GET /api/transactions/export`). **Phase 1 is now fully complete** — every route `handlers/` has, `internal/api` has too. See `.claude/rules/csv-import.md`.
- `2026-10-05`: Phase 2 (the React SPA) under way in `client/` — routing, auth context (in-memory access token, silent refresh on load), and a working first pass of all four main pages (Dashboard, Transactions, Categories, Settings), each wired to its `internal/api` counterpart via `src/hooks/` (TanStack Query). A real-browser smoke test (register → use every page → dark mode → mobile viewport) caught two backend bugs invisible to `go test`: array fields serializing as JSON `null` instead of `[]` (crashed any empty-state page) and the refresh-token cookie's hardcoded `SameSite=None` silently getting dropped by the browser in local HTTP dev (broke every hard reload). Both fixed server-side; see the new `.claude/rules/json-api-conventions.md` for the conventions that came out of it. `internal/api` also gained `theme` on `/api/me`'s user object (the client needs it on bootstrap, not just from `/api/settings`) and `hasIncome` on the dashboard's balance object (an ambiguity `spentPct` alone couldn't resolve). See `CLAUDE.md`'s "Migration in Progress" section for what's a known simplification versus the HTML side (no mobile gesture polish yet, no CSV import/export UI).
- `2026-10-05`: `internal/api` gained forgot/reset-password (`/api/forgot-password`, `/api/reset-password`) and email verification (`/api/verify-email`) — a gap from the earlier Phase 1 auth commit, filled before Phase 2's auth pages need them. `verify-email` is `POST` with the token in the body rather than `GET` with it in the query string, since the client's own `/verify-email` route (not this endpoint) is what the emailed link points at. See `.claude/rules/email-verification.md`.
- `2026-10-05`: `internal/api` gained `/api/settings` and its mutations (profile, email, resend-verification, password, theme, session revoke/revoke-others, account deletion) — Phase 1 of the Gin/React migration is now functionally complete except CSV import/export (deferred, no JSON design yet). `deleteAccountHandler`/`deleteAccount` duplicate `handlers/settings_handlers.go`'s identically, function names included; see `.claude/rules/account-deletion.md` and `.claude/rules/auth-sessions.md`'s new "Gin/JSON side" section.
- `2026-10-05`: `internal/api` gained `/api/dashboard` (`GET`), mirroring `handlers/report_handlers.go`'s pie/bar aggregation (`buildPieData`, `buildBarSeries`, same constants) but returning raw numbers instead of pre-formatted comparison sentences or `template.JS`-wrapped JSON — see `.claude/rules/dashboard.md`. Also adds `headerBalance` to the response (the nav-widget balance, always the real current month) since a JSON client has no layout the server renders it into. Settings `/api` endpoints still TODO.
- `2026-10-05`: `internal/api` gained `/api/transactions` (`GET`/`POST`/`PATCH /:id`/`DELETE /:id`) with the same filter/month-scope/paging semantics as `handlers/txn_*.go` — `transaction_query.go` duplicates `req_month.go`/`req_filters.go`/`req_paging.go`'s value objects minus the `HX-Current-URL`-reading machinery a JSON client has no use for (see `.claude/rules/req-value-objects.md`). CSV import/export endpoints still TODO (deferred, not yet designed). Dashboard/settings `/api` endpoints still TODO.
- `2026-10-05`: `internal/api` gained `/api/categories` (`GET`/`POST`/`PATCH /:id`/`DELETE /:id`), mirroring `handlers/category_handlers.go`'s business rules (default-category slug/rename/delete restrictions, swatch validation, Other-category reassignment on delete) exactly, collapsed into one `PATCH` instead of the HTML side's `.../color` + `.../name` split. Transactions/dashboard/settings `/api` endpoints still TODO.
- `2026-10-05`: Phase 1 of the Gin/React migration started — `internal/api` (new package, Gin) adds a JSON route tree under `/api/*` running alongside the existing Chi app in the same process (`cmd/server/main.go` dispatches by path prefix). Landed: stateless JWT access tokens (`internal/auth/jwt.go`) and `/api/register|login|refresh|logout|me`, reusing `internal/auth/session.go`'s existing token as the refresh token rather than adding a second revocable-token mechanism. `JWT_SECRET` (required) and `CORS_ALLOWED_ORIGINS` (optional) added to `internal/config`. See `CLAUDE.md`'s "Migration in Progress" section.
- `2026-10-05`: repo converted into a monorepo (`server/` + `client/`), start of a Chi→Gin / html-template→React migration. All Go code `git mv`'d from the repo root into `server/` with no behavior change (build/vet/tests pass identically); `render.yaml` gained `rootDir: server`. `client/` scaffolded fresh (Vite + React + TypeScript + Tailwind v4 + pnpm workspace), currently a placeholder with no real pages. Every `paths:` frontmatter entry and path reference across `.claude/rules/*.md`, `backend.md`, and `frontend.md` gained a `server/` prefix to match the move — the rules still describe the *current* Chi/html-template backend, not a future Gin/JSON one. Root `CLAUDE.md` and `README.md` rewritten for the monorepo layout. See `CLAUDE.md`'s "Migration in Progress" section for what's actually done vs. still pending.
- `2026-10-05`: bank-email auto-tracking feature removed in full — `server/internal/bankmail`, `server/internal/classify`, `server/internal/inbound`, `server/internal/inboxproc`, and `emailworker/` deleted; `bank_emails`, `category_hints`, `bank_accounts` tables and `transactions.source`/`bank_email_id`/`users.inbox_token` columns dropped via a new migration; `other_income` category kept. Repository Map entry above updated to match. `.claude/rules/email-ingestion.md` deleted; `deployment.md` and `categories.md` trimmed.
- `2026-10-04`: `CLAUDE.md` split into `.claude/context/` (this directory), `.claude/rules/`, and `.claude/skills/`; this `README.md` added as the context directory's index.
- `2026-10-04`: `backend.md` added — a single comprehensive backend reference (routes, schema, auth/authorization, env vars, deploy, known gaps, agent playbooks).
- `2026-10-04`: `frontend.md` added — a single comprehensive frontend reference (templates, static assets, the render pipeline, theming, htmx, charts, mobile nav), the browser-facing counterpart to `backend.md`.
- `2026-10-04`: `overview.md`, `stack.md`, and `request-routing.md` removed — their content was already folded into `backend.md`/`frontend.md` (product description, tech stack, request flow/package layout) or was already restated in the root `CLAUDE.md`. `.claude/context/` now holds exactly three files: this `README.md`, `backend.md`, and `frontend.md`.

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
    and not yet linked from anywhere a real user reaches. Landed so far:
    JWT access tokens (`internal/auth/jwt.go`, stateless, 15 min TTL);
    `/api/register`, `/api/login`, `/api/refresh`, `/api/logout`,
    `/api/me` — register/login reuse `internal/auth/session.go`'s existing
    token *as* the refresh token (httpOnly cookie, `Path=/api`), rather
    than a second revocable-token mechanism; see `jwt.go`'s and
    `auth_handlers.go`'s doc comments for why; and `/api/categories`
    (`GET`/`POST`/`PATCH /:id`/`DELETE /:id`) — one `PATCH` doing what the
    HTML side splits into `.../color` and `.../name` sub-routes, since a
    JSON body has no fragment-identity reason to keep them apart.
    Transactions/dashboard/settings endpoints are still TODO. No dedicated
    context file for `internal/api` yet — one gets written once it covers
    the full route surface backend.md describes for the Chi side today.
- **`client/`** is a fresh Vite + React + TypeScript + Tailwind v4 scaffold
  (pnpm workspace) with no real pages yet — see `client/src/App.tsx`. It has
  no dedicated context file yet; one gets written once it has real pages to
  describe.
- Deploy target: `server/` on Render (`server/render.yaml`, `rootDir:
  server`), `client/` on Vercel (`client/vercel.json`) once cutover happens.
  Today only `server/` is deployed; the old single-service Chi app is still
  what's live in production.

Update this section (and the Change Log in `.claude/context/README.md`) as
each migration phase actually lands — don't let it go stale.

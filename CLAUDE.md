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

- **`server/`** right now is still the original Chi + `html/template`
  monolith, relocated as-is (see `git mv` history on this directory). Every
  path in `.claude/context/backend.md`, `.claude/context/frontend.md`, and
  `.claude/rules/*.md` is written relative to `server/` (e.g.
  `server/internal/handlers/`) to match. Treat those files as still
  describing the current Chi/html-template behavior until this note says
  otherwise — they get rewritten for Gin/JSON once that rewrite actually
  lands.
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

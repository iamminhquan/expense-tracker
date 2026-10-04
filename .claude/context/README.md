# $pend Project Context

## Purpose
- This directory is the centralized project knowledge base for AI agents working in this repository.
- Canonical architecture and background context lives here. Area-specific coding rules live in `.claude/rules/` (loaded automatically for the file being touched, via `paths:` frontmatter), and step-by-step processes live in `.claude/skills/`.
- `CLAUDE.md` at the repo root is intentionally minimal: it only routes to this directory, `.claude/rules/` and `.claude/skills/`. The common commands live in `backend.md`.

## Context Files
This directory holds exactly two subject-matter files, both comprehensive single documents rather than a pile of short topic pages:
- `backend.md`: the backend source-of-truth — product description, tech stack, the full route surface, package layout, the data model, auth/authorization, env vars, deploy notes, known gaps, and agent playbooks/safe-edit rules. It is the file to update first when backend behavior changes.
- `frontend.md`: the browser-facing source-of-truth — templates, static assets (CSS/JS), the render pipeline, theming, htmx conventions, CSRF's client half, the dashboard/charts, and mobile navigation. It is the file to update first when frontend behavior changes.

Both overlap the `.claude/rules/*.md` files by design (same facts, read in one place here instead of jumping file to file), and both cross-reference each other and this README. Where a context file and a rule disagree, the rule is the source: fix the context file to match it, and fix both to match the code.

## Routing Rule
- For backend questions (routes, schema, auth, env vars, deploy, known gaps, task playbooks), read `backend.md`.
- For frontend questions (templates, static assets, the render pipeline, theming, htmx, charts, mobile nav), read `frontend.md`.
- For coding conventions and rules scoped to a specific area (templates, CSV import, the balance widget, auth/sessions, CSRF, the database, email ingestion, deployment, etc.), don't read this directory — `.claude/rules/*.md` load automatically based on the file being touched, via their own `paths:` frontmatter.
- For committing changes or opening a pull request, read `.claude/skills/`.

## Maintenance Rule
- Any agent changing behavior or operational assumptions in this codebase MUST update the relevant file in `.claude/context/` or `.claude/rules/` before finishing.
- Update context when changing:
  - routes or the `internal/handlers` package layout
  - auth or session behavior
  - the database schema or migrations
  - environment variables
  - deployment flow (Render, Neon, or the Cloudflare email worker)
  - architectural boundaries between packages
  - important conventions or source-of-truth files

## Repository Map
- `cmd/server/`: the entrypoint; wires `handlers.Deps` and starts the server.
- `internal/handlers/`: HTTP handlers — one flat package, grouped by filename prefix (see `backend.md`'s Backend Layout section).
- `internal/web/`: `go:embed`ed templates and static assets.
- `internal/database/`: migrations and hand-written SQL queries; `internal/sqlcgen/` holds the generated bindings (never hand-edited — edit the `.sql` and regenerate).
- `internal/auth/`, `internal/csrf/`: sessions, passwords, lockout, CSRF.
- `internal/csvimport/`, `internal/bankmail/`, `internal/classify/`, `internal/inbound/`, `internal/inboxproc/`: CSV import and bank-email ingestion.
- `internal/format/`, `internal/i18n/`, `internal/txnrule/`, `internal/pgval/`: shared helpers (display formatting, category names, transaction limits, pgtype wrappers).
- `emailworker/`: the Cloudflare Email Worker — a second deploy target, not touched by a normal `git push`.
- `.claude/rules/`: path-scoped coding rules, one file per area.
- `.claude/skills/`: step-by-step processes (committing changes, opening a pull request).

## Documentation Trust Rule
- Do not assume the root `README.md` is fully current.
- Prefer code and the files in `.claude/context/` and `.claude/rules/` when documentation conflicts.

## Change Log
- `2026-10-04`: `CLAUDE.md` split into `.claude/context/` (this directory), `.claude/rules/`, and `.claude/skills/`; this `README.md` added as the context directory's index.
- `2026-10-04`: `backend.md` added — a single comprehensive backend reference (routes, schema, auth/authorization, env vars, deploy, known gaps, agent playbooks).
- `2026-10-04`: `frontend.md` added — a single comprehensive frontend reference (templates, static assets, the render pipeline, theming, htmx, charts, mobile nav), the browser-facing counterpart to `backend.md`.
- `2026-10-04`: `overview.md`, `stack.md`, and `request-routing.md` removed — their content was already folded into `backend.md`/`frontend.md` (product description, tech stack, request flow/package layout) or was already restated in the root `CLAUDE.md`. `.claude/context/` now holds exactly three files: this `README.md`, `backend.md`, and `frontend.md`.

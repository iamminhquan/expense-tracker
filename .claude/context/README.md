# $pend Project Context

## Purpose
- This directory is the centralized project knowledge base for AI agents working in this repository.
- Canonical architecture and background context lives here. Area-specific coding rules live in `.claude/rules/` (loaded automatically for the file being touched, via `paths:` frontmatter), and repeatable workflows live in `.claude/skills/`.
- `CLAUDE.md` at the repo root is intentionally minimal and generic: it describes how `.claude/` is organized and names no individual context file, rule, or skill, so adding one never requires editing it.

## Context Files
- One file per package, named after the package's directory: `<package>.md` describes `<package>/` (`server.md` ↔ `server/`, `client.md` ↔ `client/`).
- Each is a single comprehensive document rather than a pile of short topic pages: what the package does, its stack and layout, its conventions, how to run, test, build and deploy it, its known gaps, and playbooks/safe-edit rules for agents. It is the file to update first when that package's behavior changes.
- They overlap the `.claude/rules/*.md` files by design (same facts, read in one place here instead of jumping file to file), and cross-reference each other and this README. Where a context file and a rule disagree, the rule is the source: fix the context file to match it, and fix both to match the code.
- A new top-level package gets its own `<package>.md` here, following the same shape.

## Routing Rule
- Working in a package: read `.claude/context/<package>.md` for that package.
- Coding rules for a specific area: don't go looking for them. `.claude/rules/*.md` load automatically for the file being touched, via their own `paths:` frontmatter.
- A repeatable workflow (committing, opening a pull request, writing a certain kind of code, and so on): `.claude/skills/` holds one directory per skill, and each skill is offered by its own `description:`. Nothing here lists them.

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
- Adding a rule or a skill needs no edit to `CLAUDE.md` or this README: a rule is found through its `paths:`, a skill through its `description:`. Keep both of those accurate instead.
- Don't enumerate the individual rules or skills in prose anywhere (context files, other rules, skills). A list like that goes stale the next time one is added. Point at the directory, or link the one specific file that explains the fact at hand.

## Repository Map
- `server/`: the Go/Gin JSON API. Its internal package layout is in `server.md`'s Backend Layout section.
- `client/`: the React/Vite SPA, a standalone pnpm package with its own lockfile. Its `src/` layout is in `client.md`'s Frontend Layout section.
- `.github/workflows/ci.yml`: the pre-merge gate. It runs each package's own checks, the same ones in each context file's Setup and Run section.
- `.claude/context/`: this directory.
- `.claude/rules/`: path-scoped coding rules, one file per area, each with its own `paths:` frontmatter.
- `.claude/skills/`: repeatable workflows, one directory per skill, each with its own `description:` frontmatter.

## Documentation Trust Rule
- Do not assume the root `README.md` is fully current.
- Prefer code and the files in `.claude/context/` and `.claude/rules/` when documentation conflicts.

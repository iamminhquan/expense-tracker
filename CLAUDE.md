# $pend

`$pend` is a monorepo of independent packages, one per top-level directory (today a Gin JSON API in `server/` and a React SPA in `client/`), talking over `/api/v1/*` with JWT auth. The project knowledge lives under `.claude/`; read only what the task needs.

@.claude/context/README.md

- `.claude/context/<package>.md` — background for the package you're working in, named after its directory (`server/` → `server.md`). Read it before non-trivial work there.
- `.claude/rules/` — coding rules, each scoped to files by its `paths:` frontmatter and loaded on its own when you touch a matching file.
- `.claude/skills/` — repeatable workflows, each offered by its own `description:`.

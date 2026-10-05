---
paths:
  - "server/render.yaml"
  - "server/.env.example"
  - "server/internal/config/**"
  - "server/cmd/server/main.go"
---

# Deployment

## Render

- `server/render.yaml` describes a free Render web service on Render's native Go runtime (no Dockerfile). Postgres is on Neon, not Render's own free tier, which is deleted after 30 days.
- `autoDeploy` is on for `main`. A schema change ships with the same push because the server migrates at startup.
- `DATABASE_URL` must be Neon's *direct* (non-`-pooler`) connection string: golang-migrate takes a session-level advisory lock the pooled endpoint doesn't support.
- `rootDir: server` makes Render `cd` into `server/` before running `buildCommand`/`startCommand` — this repo is a monorepo (`server/` + `client/`, see `CLAUDE.md`), not a single Go module at the root anymore. Build the binary into, and start it from, `server/` (not the repo root). Migrations are read from a path relative to that working directory even though templates and static assets are embedded.
- `JWT_SECRET` and `CORS_ALLOWED_ORIGINS` are Phase 1 additions for the Gin/React migration (`internal/auth/jwt.go`) — `JWT_SECRET` is required with no fallback, same as `DATABASE_URL`.

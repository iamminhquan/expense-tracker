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
- Build the binary into, and start it from, the repo root. Migrations are read from a path relative to the working directory even though templates and static assets are embedded.

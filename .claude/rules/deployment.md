---
paths:
  - "render.yaml"
  - "emailworker/**"
  - ".env.example"
  - "internal/config/**"
  - "cmd/server/main.go"
---

# Deployment

## Render

- `render.yaml` describes a free Render web service on Render's native Go runtime (no Dockerfile). Postgres is on Neon, not Render's own free tier, which is deleted after 30 days.
- `autoDeploy` is on for `main`. A schema change ships with the same push because the server migrates at startup.
- `DATABASE_URL` must be Neon's *direct* (non-`-pooler`) connection string: golang-migrate takes a session-level advisory lock the pooled endpoint doesn't support.
- Build the binary into, and start it from, the repo root. Migrations are read from a path relative to the working directory even though templates and static assets are embedded.

## The Email Worker

- `emailworker/` is a second deploy target that `git push` does not touch: a Cloudflare Email Worker (an Email Routing catch-all on `in.<domain>` forwards bank email into it).
- Deploy it by running `npx wrangler deploy` from inside that directory.
- It lives in this repo because it shares a signing and payload contract with the inbox handler. Nothing automates the deploy, so change it and forget to deploy and email silently keeps hitting the old version.

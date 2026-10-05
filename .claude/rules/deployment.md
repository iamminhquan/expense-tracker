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
- `rootDir: server` makes Render `cd` into `server/` before running `buildCommand`/`startCommand` — this repo is a monorepo (`server/` + `client/`, see `CLAUDE.md`), not a single Go module at the root. Build the binary into, and start it from, `server/` (not the repo root). Migrations are read from a path relative to that working directory.
- `JWT_SECRET` signs/verifies access tokens (`internal/auth/jwt.go`) and is required with no fallback, same as `DATABASE_URL`. `CORS_ALLOWED_ORIGINS` is the comma-separated list of origins (the Vercel domain `client/` deploys to) the API accepts credentialed cross-origin requests from -- empty means none, which is wrong once `client/` has a real deployment to allow.
- `APP_BASE_URL` must be **`client/`'s own domain (the Vercel URL), not this Render service's**: password-reset/verify-email links point at `client/`'s `/reset-password`/`/verify-email` routes, which this API has no handler for. Setting it to the Render URL by mistake sends a working email with a link to nowhere.
- `client/`'s own deploy is Vercel, not Render -- see `client/vercel.json` and its own `VITE_API_BASE_URL` env var (the Render URL above). The two services deploy independently; a push to `server/` doesn't redeploy `client/` and vice versa.

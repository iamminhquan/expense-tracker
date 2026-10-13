# Setting up, or re-wiring, the three services

Each service needs a URL from another one, so the order matters:

1. **Neon** exists first. Render's `DATABASE_URL` comes from it.
2. **Render** is created, or already exists. That fixes the API origin, `https://<service>.onrender.com`.
3. **Vercel** is created with `VITE_API_BASE_URL` = that origin. That fixes the client origin, `https://<project>.vercel.app`.
4. **Back to Render**: `CORS_ALLOWED_ORIGINS` and `APP_BASE_URL` = the client origin. Saving them redeploys Render.

Use the stable production domain (`<project>.vercel.app`, shown under the project's Domains), not a per-deployment URL with a hash in it. Write origins as `https://host`, with no trailing slash and no path. CORS compares them as exact strings.

## Cutover from the monolith

Before the split, the whole app (a Chi + `html/template` server) ran alone on the same Render service and the same Neon database. That service and database still exist and hold real data. The cutover re-points them; it doesn't recreate them. Go step by step, checking each one before the next.

### 0. Find out where things stand

- `run_sql` on `production`: `SELECT version, dirty FROM schema_migrations`. Compare with the highest number in `server/internal/database/migrations/`. If production is behind, the new server hasn't started successfully against it yet, and the gap is what the first deploy will run.
- Ask the user for the Render service URL, and what Render → Events shows: which commit is live, and whether recent deploys failed. A failed deploy log saying `configuration: ...` means a missing env var (`JWT_SECRET` didn't exist in the monolith).

### 1. Back up, and rehearse the pending migrations

Follow [migrations.md](migrations.md): a `pre-cutover-<YYYYMMDD>` branch, then a rehearsal of the gap on a throwaway branch. This first deploy runs several migrations against data written by the old app, so the rehearsal is worth more here than on any later release.

### 2. Repair the Render service settings

Walk the user through Render → the service → Settings, comparing each field with `server/render.yaml`. These are the ones the monolith had set differently:

| Setting | Must be | Why it may differ |
|---|---|---|
| Branch | `main` | the Blueprint was written when the default branch was `master`, which no longer exists on `origin`. A service still tracking it never auto-deploys |
| Root Directory | `server` | the monolith lived at the repo root |
| Build Command | `go build -o server ./cmd/server` | |
| Start Command | `./server` | |
| Health Check Path | `/healthz` | |
| Blueprint file path (only if the service is Blueprint-managed) | `server/render.yaml` | the file moved from the repo root into `server/`, so a Blueprint still looking at `render.yaml` finds nothing |

Then Environment. Go through `render.yaml`'s `envVars` one at a time:

- Fixed values (`GO_VERSION`, `GIN_MODE`, `SECURE_COOKIES`, `SESSION_COOKIE_NAME`): copy them from the file.
- `DATABASE_URL`: keep the existing one if its host has no `-pooler`. Otherwise the user copies the **direct** string from the Neon console.
- `JWT_SECRET`: new. The user runs `openssl rand -base64 32` and pastes the output in themselves.
- `CORS_ALLOWED_ORIGINS`, `APP_BASE_URL`: the Vercel origin, so these wait until step 3. `APP_BASE_URL` in the monolith was this Render URL itself, which is now wrong: email links would point at an API with no `/reset-password` page.
- `TRUSTED_PROXIES`: read its comment in `render.yaml`, and the check it describes after the deploy.
- `BREVO_API_KEY`, `MAIL_FROM`: likely already there. Keep them.

Variables in the dashboard that `render.yaml` doesn't list are monolith leftovers. List them for the user, and delete them only on their say-so.

### 3. Create the Vercel project

The user does this in Vercel: Add New → Project → import `iamminhquan/expense-tracker`. Then:

- Root Directory: `client`. Framework, build command and output come from `client/vercel.json`, and the pnpm version from `packageManager` in `client/package.json`.
- Production Branch: `main`.
- Environment Variables → `VITE_API_BASE_URL` = the Render origin, for Production, with type **Config**, not Secret. Every `VITE_*` value is inlined into the JavaScript the browser downloads, so Vercel refuses to save one as Secret ("Remove the public framework prefix…"). It's a public URL anyway.
- Optional: Settings → Git → turn on skipping builds when nothing under the root directory changed, so a server-only push doesn't rebuild the client.

Deploy. If `VITE_API_BASE_URL` was added after the first build, or scoped to Preview only, that build has none. The client then calls `/api/v1/*` on Vercel itself, which answers 405, and login fails. `smoke.sh`'s bundle check catches this. The fix is Deployments → the latest → Redeploy, since a saved env var never reaches an existing build. Take the production domain and set `CORS_ALLOWED_ORIGINS` and `APP_BASE_URL` on Render (step 2's remaining two).

### 4. Deploy the server

If saving the env vars didn't already start a deploy, have the user click Manual Deploy → Deploy latest commit. Watch it the way the routine release does: `schema_migrations` reaching the repo's highest version, then `/healthz`.

### 5. Verify and record

- Run `scripts/smoke.sh`, then have the user log in on the Vercel URL and reload the page.
- Check the keep-alive cron still pings `<render-origin>/healthz` every ~10 minutes. In the monolith it may have pinged `/`, which now answers 404.
- Tell the user the Render URL no longer serves pages: anyone with it bookmarked needs the Vercel one.
- Write both origins into the Deploy Notes of `.claude/context/server.md` and `client.md`, and keep the backup branch until the user is satisfied.

## Recreating a service from scratch

- **Neon**: a new project in `aws-ap-southeast-1` (Singapore, the same region as the Render service). Postgres version: match the current project's (18) unless the user decides otherwise. The schema builds itself on the server's first start. Take the direct connection string. Data comes only from a restore or a dump, never from the migrations.
- **Render**: New → Blueprint → this repo, Blueprint path `server/render.yaml`. Render then asks for every `sync: false` variable. Leave `CORS_ALLOWED_ORIGINS` and `APP_BASE_URL` for after Vercel exists.
- **Vercel**: as in step 3 above.

## Known issues to warn about

- **Third-party cookie.** `*.vercel.app` and `*.onrender.com` are different sites (both are on the Public Suffix List), so the refresh cookie Render sets is a third-party cookie to the page on Vercel. That is why it is `SameSite=None; Secure`. Browsers that block third-party cookies, Safari above all and so every iPhone, drop it, and those users are logged out on every reload. The cure is a decision for the user, not a deploy step: put both behind one custom domain (`app.` + `api.` on the same registrable domain makes the cookie first-party), or proxy `/api/*` through a Vercel rewrite so the API is same-origin. A proxy changes the client IP Render sees, so it changes `TRUSTED_PROXIES` too. Either way it touches code, `vercel.json` and the docs.
- **Vercel Deployment Protection** (on by default) puts generated URLs, such as `expense-tracker-<team>.vercel.app` and every hashed deployment URL, behind a Vercel login. The owner, being logged in, sees the app fine. Everyone else, and `curl`, gets a 302 to `vercel.com/sso-api`. The public origin must be a domain added under Settings → Domains for Production. A `<name>.vercel.app` added there counts. That domain is the one to put in `CORS_ALLOWED_ORIGINS` and `APP_BASE_URL`.
- **Vercel preview deployments** have their own origins, which are not in `CORS_ALLOWED_ORIGINS`, so a preview shows the UI but can't log in. That is expected. Don't add preview origins to production CORS.
- **Cold starts**: a free Render instance sleeps after 15 idle minutes, and the first request then takes 30–60 s. The keep-alive cron hides this. A smoke test right after a quiet period may need the retries in `smoke.sh`.

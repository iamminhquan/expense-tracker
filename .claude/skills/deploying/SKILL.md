---
name: deploying
description: Ship $pend to production — `server/` on Render, `client/` on Vercel, Postgres on Neon — and prove it works afterwards. Covers a routine release (pre-flight checks, env vars before code, merging to `main`, watching both deploys, smoke tests), keeping a migration safe on the production database (Neon backup branch, rehearsal, expand/contract), and the one-time setup or cutover of the three services. Use whenever the user wants to deploy, release, ship, go live, push to prod, merge something "and deploy it", set or change a production env var, check whether a deploy worked or what is live, or asks how a migration will reach the production database — even when they name only one of Render, Vercel or Neon.
---

# Deploying $pend

Three services, each wired to the others by an env var. Nothing deploys by hand: a push to `main` is the release.

```
browser ──► Vercel (client/)  ── VITE_API_BASE_URL ──►  Render (server/)  ── DATABASE_URL ──►  Neon (Postgres)
              ▲                                            │
              └──── CORS_ALLOWED_ORIGINS, APP_BASE_URL ────┘
```

- **Render** builds `server/` (`rootDir: server`, native Go runtime, no Dockerfile) on every push to `main` that touches `server/`, and the binary applies pending migrations at startup, before it listens. `server/render.yaml` is the source of truth for its settings; read its comments rather than restating them.
- **Vercel** builds `client/` (Root Directory `client`, `client/vercel.json`). `VITE_API_BASE_URL` is inlined at **build** time, so changing it means a redeploy.
- **Neon** project "Expense Tracker App" (`bitter-wildflower-45850102`, `aws-ap-southeast-1`, Postgres 18). The production branch is named `production`, database `neondb`. The free plan keeps only **6 hours** of history, so point-in-time restore is no safety net for a migration you notice the next morning. A named backup branch is.
- The live URLs belong in the Deploy Notes of `.claude/context/server.md` (Render) and `.claude/context/client.md` (Vercel). If they aren't there, ask the user and write them in. Everything below needs them.

## How to work

Do the steps that have a tool yourself: `gh`, `curl`, `psql`, `go`, `pnpm`, and the Neon MCP tools. The Vercel MCP (the `vercel` plugin, or the claude.ai Vercel connector) reads the project (`expense-tracker`, `prj_vVkyIPB6ZCzGuFLB36x5gdJyDI0g`, team `iamminhquans-projects`), but only if the user granted that team at the OAuth consent screen. Without it, every project-level call answers 403 `Trying to access resource under scope`. Say so once and fall back to `curl` and the dashboard; don't retry. Render has no CLI or API access here. Dashboard steps go to the user as precise instructions: which page, which field, what value. Then check the result yourself, because the user can't easily see whether a value took effect.

Ask before anything that reaches production or can't be undone: merging to `main` (it **is** the deploy), any SQL that writes to the `production` branch, deleting or restoring a Neon branch, and removing an env var. Read-only checks on production (`SELECT` from `schema_migrations`, `curl`) need no confirmation.

Keep secrets out of the transcript. The user generates `JWT_SECRET` themselves (`openssl rand -base64 32`) and pastes it straight into Render. The production `DATABASE_URL` goes from the Neon console straight into Render; don't fetch it with `get_connection_string`. A new Neon branch shares its parent's passwords, so reset the role's password on a throwaway branch *before* fetching its connection string (see the rehearsal steps). Keep that string in a shell variable for one command; it never goes into a file in the repo.

## Routine release

### 1. Work out what is shipping

```bash
git fetch origin
git log --oneline origin/main..HEAD                      # or the PR's commits: gh pr view <n> --json commits
git diff --stat origin/main...HEAD -- server client
```

From the diff, note four things. Each one adds a step below.

| In the diff | Means |
|---|---|
| anything under `server/` | Render redeploys |
| anything under `client/` | Vercel redeploys |
| `server/internal/database/migrations/` | a migration will run on production → step 3 |
| `server/.env.example`, `server/render.yaml` `envVars`, a new `import.meta.env.VITE_*` | an env var to set **before** merging → step 2 |

### 2. Set new env vars first

`config.Load()` refuses to start without a required variable, so code that needs a new one, merged before the variable exists, fails Render's health check. Render then keeps the old instance serving, but nothing new is live, and an env var nobody set is easy to miss in the logs. Have the user add it in the dashboard first. Render: service → Environment. Vercel: Project → Settings → Environment Variables, scoped to Production. A `VITE_*` variable changed on an already-deployed client needs a Vercel redeploy to take effect.

### 3. Migrations: back up, rehearse, check compatibility

Do this whenever step 1 found a new migration file. The details and the reasons are in [references/migrations.md](references/migrations.md). In short:

1. Read every new `*.up.sql` and judge whether the **currently live code** still works against the new schema. Render starts the new instance (which migrates) while the old one still serves traffic. If the new one then fails its health check, the old code keeps running against the new schema indefinitely. Adding things is safe. Dropping, renaming, or tightening a constraint the old code violates is not. Split those across two releases.
2. Create a Neon branch from `production` named `pre-release-<NNNNNN>-<YYYYMMDD>`, where `<NNNNNN>` is the highest migration about to run. It is instant and free, and it is the restore point.
3. Rehearse: run the new server against a second throwaway branch so the migrations apply there first. Confirm `schema_migrations` reaches the new version with `dirty = false`. Then delete the throwaway, after asking.

### 4. Pre-flight

CI (`.github/workflows/ci.yml`) is the gate. On a PR, `gh pr checks <n>` must be all green. Pushing straight to `main` skips that gate, so run CI's checks locally for each package the diff touches. `server/`, from inside `server/`: `go build ./...`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `go test ./...`. `client/`, from inside `client/`: `pnpm lint`, `pnpm test`, `pnpm build`. Don't merge on red.

### 5. Merge (= deploy)

Confirm with the user, then merge the PR the way this repo does, with a merge commit: `gh pr merge <n> --merge`. Opening the PR itself is the `creating-pr` skill's job. Note the merge commit's SHA; every check below looks for it.

### 6. Watch both deploys

- **Vercel** reports to GitHub: `gh api repos/{owner}/{repo}/commits/<sha>/status --jq '.statuses[] | {context, state, target_url}'` (also `gh pr checks` on the merged PR). Wait for `success`.
- **Render** may not report to GitHub. Poll `/healthz` (a free instance asleep takes 30–60 s to wake), and when the release had a migration, poll `SELECT version, dirty FROM schema_migrations` on `production` until it shows the new version. The migration running is the surest sign the new binary started. With no migration, ask the user to confirm Render → Events shows "Deploy live" for this commit. Render's build output appears only in its dashboard logs; if the deploy fails, ask the user to paste the failing lines.

### 7. Smoke test

```bash
bash .claude/skills/deploying/scripts/smoke.sh <render-origin> <vercel-origin>
```

It is read-only and checks: `/healthz`; that the API answers in its JSON envelope; CORS for the Vercel origin, with credentials; the SPA loading and deep links surviving a reload (the rewrite in `vercel.json`); and that the built bundle really points at the Render origin (a forgotten or mistyped `VITE_API_BASE_URL` fails here and nowhere else). Any `FAIL` stops the release report. Fix it or tell the user.

`curl` can't check a real browser's cookie policy. When the release touched auth, cookies, CORS or env vars, ask the user to log in on the Vercel URL and **reload the page**. Staying logged in proves the cross-site refresh cookie round-trips. Ask them to try Safari/iPhone too, which blocks third-party cookies most aggressively (see the setup reference's known issues).

### 8. Report

Say what went live (SHA, which of server/client), the migration version before and after, the smoke results, and the name of the backup branch, if one was made. Remind the user it can be deleted once the release has proven itself. Neon's free plan caps the number of branches.

## When something goes wrong

- **Code bug, schema fine:** Render → the previous deploy → Rollback, or Vercel → Deployments → the previous one → Instant Rollback. Both are dashboard actions for the user. Then fix forward with a normal release. A code rollback after an *additive* migration is safe. After a destructive one it is not; see the migrations reference.
- **Render deploy fails on startup:** a missing env var (`configuration: ...` in the log), a dirty migration (`Dirty database version N`), or a failed migration. [references/migrations.md](references/migrations.md) has the dirty-version recovery.
- **CORS errors or "logged out on reload" in the browser:** check `CORS_ALLOWED_ORIGINS` exactly (scheme, no trailing slash) and `SECURE_COOKIES=true`, then rerun the smoke test.
- **Data damaged by a migration:** restore `production` from the `pre-release-*` branch. Writes made since the backup are lost, so this is the user's call. Prefer a forward fix when the damage is small.

## First-time setup, or a service that needs re-wiring

Read [references/setup.md](references/setup.md). It covers creating or repairing each service, the order the env vars must go in (each service needs the other's URL), and the cutover from the old monolith that used to run alone on Render.

## Keep the docs true

If a deploy changes how deployment works (a new env var, a changed Render or Vercel setting, a live URL), update `server/render.yaml`'s comments, `server/.env.example`, `.claude/rules/deployment.md`, and the Deploy Notes in `.claude/context/server.md` / `client.md` in the same change, as `.claude/context/README.md` requires. When you learn something about this deploy that would have saved time, add it to this skill too.

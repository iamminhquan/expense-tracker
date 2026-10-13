# Migrations on production

The server runs `migrate.Up()` at startup (`runMigrations` in `server/cmd/server/main.go`), before it listens. Nobody runs a migration against production by hand: merging a migration file to `main` *is* running it. Everything here happens before that merge, or after it if something went wrong.

## Why "will the old code survive this?" is the first question

A Render deploy starts the new instance next to the old one. The order is:

1. The new binary starts, migrates the shared Neon database, and starts listening.
2. Render checks `/healthz` on it. Only when that passes does traffic move over and the old instance stop.

So for a while, often seconds and sometimes much longer, the **old code serves real users against the new schema**. If step 2 fails (a bug, a missing env var that is read after migrating, a crash), the deploy is marked failed and the old code keeps serving against the new schema indefinitely. A code rollback lands in the same situation: old code, new schema, because nothing runs down migrations.

| Change | Old code against new schema | Do |
|---|---|---|
| New table, new nullable column, new column with a default, new index, a seed `INSERT … WHERE NOT EXISTS` | fine | ship in one release |
| `DROP COLUMN`/`TABLE`, `RENAME`, change a column type | breaks queries that name it | two releases: first stop using it in code, then drop it |
| `SET NOT NULL`, a new `CHECK`/`UNIQUE` the old code's writes can violate | old code's inserts fail | backfill and update the code first, then add the constraint |
| `UPDATE`/`DELETE` of existing rows | depends: read it | treat as data loss risk; the backup branch matters most here |

The default categories have their own rule: `.claude/rules/categories.md` and migrations 000006/000008/000014 explain why they are only ever updated in place.

## Backup branch

Neon's free plan keeps 6 hours of history, so make an explicit restore point:

- `list_branches` (project `bitter-wildflower-45850102`) → the id of `production`.
- `create_branch` with `parent_id` = that id, `name` = `pre-release-<NNNNNN>-<YYYYMMDD>`, and `no_compute: true`. It's a restore point, not something to connect to, and without a compute it costs nothing while it sits there.

Creating a branch is instant (copy-on-write) and doesn't touch `production`, so it needs no confirmation. Deleting one later does.

## Rehearsal

Run the real migration code against a copy of production data:

1. `create_branch` from `production`, named `rehearse-<NNNNNN>`, **with** a compute.
2. `reset_postgres_role_password` for `neondb_owner` on **the rehearsal branch**, before anything else. A branch inherits its parent's roles *with the same passwords*, so the connection string of a fresh branch carries production's password. The reset is scoped to that one branch and leaves production's untouched.
3. `get_connection_string` for that branch. If the host contains `-pooler`, remove it: golang-migrate takes a session-level advisory lock that the pooled endpoint doesn't support, the same reason Render's `DATABASE_URL` is the direct one.
4. From `server/`, build first, then start the binary against the branch and wait for `/healthz`. That proves the migrations applied, since they run before listening. Use `go build`, not `go run`: compiling alone can outlast a minute of polling and look like a hung migration.

   ```bash
   go build -o "$SCRATCH/server-rehearse" ./cmd/server
   DATABASE_URL="$REHEARSE_URL" JWT_SECRET=rehearsal PORT=18080 "$SCRATCH/server-rehearse"
   ```

   Run it in the background, poll `curl -s -o /dev/null -w '%{http_code}' localhost:18080/healthz` until it answers 200, read the log for `apply migrations:` errors, then stop it. A variable set on the command line beats `server/.env` (godotenv never overrides), so the local `.env` can't redirect this to another database.
5. `run_sql` on the rehearsal branch: `SELECT version, dirty FROM schema_migrations`. Expect the highest migration number in `server/internal/database/migrations/` with `dirty = false`. For a data migration, also spot-check the rows it changed.
6. Ask, then `delete_branch` the rehearsal branch.

A rehearsal catches SQL that fails on real data (a constraint that existing rows violate, a duplicate the new `UNIQUE` index trips over) that CI's empty database never sees.

## Dirty version

golang-migrate sets `dirty = true` before running a file and clears it after. If a file fails, the flag stays set, and every later start fails with `Dirty database version N. Fix and force version.`, so Render can't deploy anything until it is cleared.

1. `SELECT version, dirty FROM schema_migrations` on `production`.
2. Work out how much of migration N actually applied. A file executed as one multi-statement batch usually rolls back as a whole, but statements that can't run in a transaction (`CREATE INDEX CONCURRENTLY`) or an explicit `BEGIN/COMMIT` in the file break that. Inspect the objects the file touches (`describe_table_schema`, or `\d` in `psql`) rather than assume.
3. Bring the schema back to exactly "N-1 applied" or "N applied", then set the row to match, e.g. `UPDATE schema_migrations SET version = <N-1>, dirty = false`. This writes to production: show the user the statement and get a yes first.
4. Fix the migration file (if N never applied) and release again.

## Restoring from the backup branch

Use this when a migration damaged data, not when it merely failed. A failed migration rolled back is a dirty-version problem, above. Restoring `production` from `pre-release-*` replaces its data with the snapshot, and every write since is lost (new transactions users entered). Before offering it:

- check whether a forward fix (a new migration that repairs the rows) is possible. It usually is, and it loses nothing;
- if restoring is the answer, have the user do it from the Neon console (Branches → `production` → Restore → from branch `pre-release-…`), or do it through the Neon MCP only after an explicit yes. Neon keeps the pre-restore state as a backup branch of its own, so mention that it exists.

Code that expects the newer schema will break against a restored older one. Roll back the Render deploy at the same time, and say so.

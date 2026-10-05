---
paths:
  - "server/internal/database/migrations/*.sql"
  - "server/internal/database/queries/*.sql"
  - "server/internal/sqlcgen/*.go"
  - "server/sqlc.yaml"
  - "server/internal/database/db.go"
---

# Database

## Rules

- The schema lives in `server/internal/database/migrations/` as numbered `NNNNNN_description.up.sql` / `.down.sql` pairs applied by golang-migrate.
- Hand-written queries live in `server/internal/database/queries/*.sql`. `sqlc` generates the Go bindings into `server/internal/sqlcgen/` (package `sqlcgen`, `pgx/v5` driver). Never hand-edit `server/internal/sqlcgen/`; edit the `.sql` and regenerate.
- Write migrations data-preserving and idempotent where they can be. 000006 and 000008 both `UPDATE ... IN PLACE` instead of delete-and-reinsert, because `transactions.category_id` has no `ON DELETE` clause and any account with history would break.
- Month-based queries (transactions list, dashboard totals) take explicit `[from, to)` date bounds computed in `server/internal/api/transaction_query.go`.

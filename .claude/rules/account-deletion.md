---
paths:
  - "server/internal/handlers/settings_handlers.go"
  - "server/internal/api/settings_handlers.go"
  - "server/internal/database/queries/users.sql"
  - "server/internal/database/queries/transactions.sql"
  - "server/internal/database/queries/categories.sql"
---

# Account deletion

Deleting an account is a hard delete (`deleteAccountHandler` / `deleteAccount` in `settings_handlers.go` -- duplicated identically, function names included, in `internal/api/settings_handlers.go` for the Gin/JSON side of the migration; see `CLAUDE.md`).

## Rules

- No `deleted_at` flag and no grace period. It would put an "is this account still alive" predicate on every query, and $pend has no billing, audit trail or support desk that a recoverable window would serve.
- `deleteAccount` removes, in one DB transaction and in this order: transactions, then the account's own categories, then the user row. Keep those steps explicit; don't lean on the `ON DELETE CASCADE` on `users`.
- Never touch the shared defaults. They carry a NULL `user_id`, so `WHERE user_id = $1` cannot reach them.
- Sessions and both token tables cascade; leave them to it.
- Gate the delete on the current password, like the email and password forms. On success redirect to `/login?deleted=1`, since no account is left to show the message to.
- Keep a CSV export link above the delete button in the danger-zone card. The history is the part anyone regrets losing.
- The email is released at once and can be re-registered (`TestDeletedEmailCanRegisterAgainAsAFreshAccount` pins it). Don't reserve it or keep a tombstone: that would keep the one piece of personal data the owner asked to be rid of.

## Why

A single cascading delete works only because `transactions.category_id` has no `ON DELETE` clause and Postgres defers that NO ACTION check to the end of the statement. That is invisible to anyone reading the delete, and one constraint change from not being true.

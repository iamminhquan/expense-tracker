---
paths:
  - "server/internal/i18n/**"
  - "server/internal/api/category_*.go"
  - "server/internal/database/queries/categories.sql"
---

# Categories

## Rules

- A default category is identified by its `slug`, **never** by its displayed name. Do not compare, group or look up a default by `name`, and do not hardcode a displayed string to find one.
- A category is either personal (created by a user, `slug` NULL, shown under the name its owner typed) or a shared default seeded by migrations (`user_id` NULL, a stable language-independent `slug`).
- A default's displayed name comes from `server/internal/i18n`: `CategoryName(slug, name)` for a row in hand, `NameForSlug` where a category is synthesised out of nothing (the dashboard pie chart's aggregate slice). A slug the map doesn't know falls back to the `name` column, so migrations keep that column populated with a usable English label. `categoryDTO`/`transactionDTO`/pie-legend entries all resolve through `CategoryName` server-side -- the client has no copy of this table and shouldn't need one just to display a row.
- Look defaults up by slug as well. The transactions search turns the typed term into slugs with `i18n.SlugsMatching` (in `transaction_query.go`) instead of matching the `name` column.
- `transactions.category_id` has no `ON DELETE` clause, on purpose: a category can't be removed while transactions reference it unless the app reassigns those rows first.
- A migration that touches a default updates it in place; never delete and reinsert it.
- Add a new default through a migration that follows the 000006 / 000008 / 000014 pattern (an idempotent insert guarded by `WHERE NOT EXISTS`, a `slug` from day one), together with its entry in `categoryNames`. See `database.md` for the schema side.
- `PATCH /api/v1/categories/:id` takes an optional `name` and/or `color` in one request -- a default category can be recolored by anyone but never renamed (`isDefault` on `categoryDTO` is what the client checks before offering that action at all).

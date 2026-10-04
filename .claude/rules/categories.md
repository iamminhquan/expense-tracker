---
paths:
  - "internal/i18n/**"
  - "internal/handlers/category_*.go"
  - "internal/handlers/req_filters.go"
  - "internal/inboxproc/**"
  - "internal/database/queries/categories.sql"
  - "internal/web/templates/categories.html"
  - "internal/web/templates/category_row.html"
---

# Categories

## Rules

- A default category is identified by its `slug`, **never** by its displayed name. Do not compare, group or look up a default by `name`, and do not hardcode a displayed string to find one.
- A category is either personal (created by a user, `slug` NULL, shown under the name its owner typed) or a shared default seeded by migrations (`user_id` NULL, a stable language-independent `slug`).
- A default's displayed name comes from `internal/i18n`: `CategoryName(slug, name)` for a row in hand, `NameForSlug` where a category is synthesised out of nothing (the pie chart's aggregate slice). A slug the map doesn't know falls back to the `name` column, so migrations keep that column populated with a usable English label.
- Look defaults up by slug as well. `GetCategoryBySlug` is how email ingestion finds `other` / `other_income`. The transactions search turns the typed term into slugs with `i18n.SlugsMatching` (in `req_filters.go`) instead of matching the `name` column.
- `transactions.category_id` has no `ON DELETE` clause, on purpose: a category can't be removed while transactions reference it unless the app reassigns those rows first.
- A migration that touches a default updates it in place; never delete and reinsert it.
- Add a new default through a migration that follows the 000006 / 000008 / 000014 pattern (an idempotent insert guarded by `WHERE NOT EXISTS`, a `slug` from day one), together with its entry in `categoryNames`. See `database.md` for the schema side.

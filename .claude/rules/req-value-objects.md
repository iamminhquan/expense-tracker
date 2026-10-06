---
paths:
  - "server/internal/api/transaction_query.go"
  - "server/internal/api/transaction_handlers.go"
  - "client/src/pages/transactions/TransactionsPage.tsx"
  - "client/src/pages/dashboard/DashboardPage.tsx"
---

# Month, filters, paging

`server/internal/api/transaction_query.go` holds the small value objects parsed out of a transactions/dashboard request: which month, which filters, which page.

## `monthScope`

- `currentMonthRange`, `monthRangeFor`, `monthLabel`, `monthOptions` (the picker's entries, built the same way for the transactions list and the dashboard) and `vietnamLocation` all live here.
- Every month window is half-open `[from, to)` anchored to `Asia/Ho_Chi_Minh`, not server UTC, with a fixed UTC+7 fallback if tzdata is missing from the runtime image.
- `monthScope` answers "one month or all of them" for the transactions list. `?month=all` resolves to bounds wide enough (`allTimeFrom` / `allTimeTo`) that the month predicate stops narrowing, so the list, the count and the export keep running the one query each already ran.
- A scope carries the spelling it arrived as (`Value`), because the client's own URL names it, and an all-time window formatted as a month would read `0001-01`.
- The dashboard is deliberately **not** a consumer of `monthScope` -- its aggregates are month-against-month, so `dashboard_handlers.go` keeps calling `monthRangeFor` directly, which treats `all` as malformed and falls back to the current month.

## `txnFilters`

- Holds search, type, category and min/max amount. The 0 sentinel means "not filtering". It also holds the nullable sqlc params the list, count, and export queries all take.
- `Sort` rides in the same object but is not a filter: it narrows nothing.
- Orders live in `sortOrders`, and the ORDER BY switches on the bound value through a pair of `CASE`s in the SQL itself. Never interpolate a column name. An unknown order matches neither and falls back to `occurred_on DESC, id DESC`.

## `pagerDTO`

- `pageSize` (10) and `newPager`, which clamps any requested page into one that exists.

## Client: the URL is the source of truth, not `useState`

- `TransactionsPage.tsx` and `DashboardPage.tsx` both keep their filters in the URL's own query string (`useSearchParams`), not local component state. This isn't just style -- a real browser test caught the bug that happens otherwise: a link to a specific month (the CSV import flow's "view results" link, a bookmark, a reload) silently reset to whatever the component's initial state happened to be, because nothing ever read the URL it landed on. See `.claude/rules/json-api-conventions.md`'s browser-testing note.
- Any new page with its own filters (CSV import's result links, say) should follow the same pattern from the start rather than needing the same fix applied after the fact.

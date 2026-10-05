---
paths:
  - "server/internal/handlers/req_*.go"
  - "server/internal/api/transaction_query.go"
  - "server/internal/api/transaction_handlers.go"
---

# Month, filters, paging

The `req_` files in `server/internal/handlers/` are small value objects parsed out of a request.

`server/internal/api/transaction_query.go` duplicates `req_month.go`'s,
`req_filters.go`'s, and `req_paging.go`'s value objects verbatim for the
Gin/JSON side of the migration (see `CLAUDE.md`), minus everything that
exists only because an htmx mutation POST carries no query string of its
own: `HX-Current-URL` reading, `scopeFromRequest`/
`filtersFromHXCurrentURL`/`pageFromRequest`, and the canonical
`?month=&page=&...` URL builders (`transactionsURL`, `exportURL`). A JSON
client already holds the filter/page state it sent and resends it, so none
of that has an equivalent here. Keep the two copies' actual filtering/
scoping/paging *behavior* identical by hand until Phase 4 deletes one of
them -- a bug fixed in one and not the other would make the old and new UI
disagree on what a filter matches.

## Shared pattern

- Parse leniently from the URL and never error on a malformed value.
- Offer a variant that reads the *originating* page's URL from the `HX-Current-URL` header (through `currentURLQuery` in `req_params.go`), because a mutation POST/PATCH/DELETE carries no query string of its own. They are `scopeFromRequest`, `filtersFromHXCurrentURL` and `pageFromRequest`.
- `req_txnform.go` is the one exception (below).

## `req_month.go`

- Holds `currentMonthRange`, `monthRangeFor`, `scopeFromRequest`, `monthLabel`, `monthOptions` (the picker's entries, built the same way for both pages that render it) and `vietnamLocation`.
- Every month window is half-open `[from, to)` anchored to `Asia/Ho_Chi_Minh`, not server UTC, with a fixed UTC+7 fallback if tzdata is missing from the runtime image.
- `txnScope` answers "one month or all of them" for the transactions list. `?month=all` resolves to bounds wide enough (`allTimeFrom` / `allTimeTo`) that the month predicate stops narrowing, so the list, the count and the export keep running the one query each already ran.
- A scope carries the spelling it arrived as, because every link the page builds must name it, and an all-time window formatted as a month reads `0001-01`.
- The dashboard is deliberately **not** a consumer of `txnScope`. Its cards and charts are month-against-month, so it keeps calling `monthRangeFor`, which treats `all` as malformed. A hand-typed `/dashboard?month=all` therefore lands on the current month. For the same reason the picker offers the `all` entry only under `ActiveNav == "transactions"`.

## `req_filters.go`

- `txnFilters` holds search, type, category and min/max amount. The 0 sentinel means "not filtering". It also holds the nullable sqlc params that both the list and count queries take, and `transactionsURL`, the canonical address pushed via `HX-Push-Url`.
- `Sort` rides in the same object but is not a filter: it narrows nothing. `Any`, `ActiveCount` and the badge leave it out, and `Sorted` is the separate predicate the create handler asks.
- Orders live in `sortOrders`, and the ORDER BY switches on the bound value through a pair of `CASE`s. Never interpolate a column name. An unknown order matches neither and falls back to `occurred_on DESC, id DESC`.

## `req_paging.go`

- `pageSize` (10) and `pager`, which clamps any requested page into one that exists.

## `req_txnform.go`

- `txnForm` is the add/edit transaction as it arrives off a request. These four values are what a row is written from, so one that won't parse is a 400, not a default that quietly banks the wrong number.
- Create and edit both read through it and both ask the same `violation` for the rules a parsed form can still break.

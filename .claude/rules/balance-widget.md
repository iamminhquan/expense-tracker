---
paths:
  - "server/internal/api/dashboard_handlers.go"
  - "client/src/components/layout/BalanceWidget.tsx"
  - "client/src/components/layout/Layout.tsx"
  - "client/src/components/layout/UserMenu.tsx"
---

# The balance widget

## Server (`dashboard_handlers.go`)

- `headerBalance` on `GET /api/v1/dashboard`'s response always reports the real current month, never the month being browsed -- unlike every other field in that response, which describes whatever month the request asked for. There is no separate endpoint for it; a JSON client that needs the widget on a page other than Dashboard still calls `/api/v1/dashboard` for it (see `Layout.tsx` below).
- It carries forward across months: what a month closes at is what the next one opens with. `MonthlyTotals` returns the carried-in figure as a third column (`carried_over`) next to the month's own two totals. Its `WHERE` reaches over the user's whole history and each column narrows through its own `FILTER`; read it carefully before changing it.
- Keep the `::bigint` around the whole subtraction in that query. Without it sqlc types the result `int32`, which overflows past 2.1 tỷ đồng.
- `newBalanceDTO` resolves the spent-percentage in Go, not the client: every case here (a month with no income, a month that overspent) needs a divide-by-zero guard, and doing it once server-side means every client agrees.
- `HasIncome` exists because `SpentPct` alone is ambiguous at zero -- "no income this month" and "income, nothing spent of it yet" both leave it at its zero value. See `.claude/rules/json-api-conventions.md`'s note on this being a real bug a browser test caught.
- Only the balance itself is cumulative. `SpentPct` and whatever sentence the client builds from it still measure the displayed month against that month's own income.

## Client (`BalanceWidget.tsx`, `Layout.tsx`)

- `Layout.tsx` calls `useDashboard()` once, at the top of the authenticated route tree, so every page shares the one cached `headerBalance` instead of each page issuing its own request -- the client-side equivalent of the widget living in a shared layout rather than each page.
- `BalanceWidget.tsx` draws a ring for `spentPct` (empty when `hasIncome` is false) beside the balance. The arc is accent-coloured and turns `danger` once spending passes the month's income; the track uses the `border` token so it stays visible on both the header and the menu's tinted panel. On desktop it sits in the header, bare, with no pill around it; on mobile `UserMenu.tsx` renders it at the top of its panel, since the mobile header has no room.
- `BalanceWidget.tsx`'s `ratioLabel` composes the "Spent X% of this month's income" / "No income this month" sentence from `spentPct` + `hasIncome` -- the thing `dashboard_handlers.go` deliberately stopped doing server-side (see `.claude/rules/dashboard.md`).
- A mutation (create/update/delete a transaction) invalidates the `['dashboard']` TanStack Query key (see `useTransactions.ts`'s `invalidateEverythingATransactionTouches`), which is what keeps the widget correct after an edit -- the client-side replacement for the old `header_balance_oob` out-of-band swap.

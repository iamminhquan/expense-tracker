---
paths:
  - "server/internal/handlers/balance_*.go"
  - "server/internal/web/templates/header_balance.html"
  - "server/internal/handlers/txn_mutate.go"
  - "server/internal/handlers/category_handlers.go"
  - "server/internal/web/templates/nav.html"
  - "server/internal/web/templates/mobile_header.html"
---

# The balance widget

## Rules

- The balance lives in one place: the `header_balance` widget (`header_balance.html`), rendered by both nav bars. There is no balance card in any page body; that partial was deleted.
- It always reports the real current month, never the month a page is browsing, because it sits in the layout above the month picker.
- It carries forward across months: what a month closes at is what the next one opens with. `MonthlyTotals` returns the carried-in figure as a third column (`carried_over`) next to the month's own two totals. Its `WHERE` reaches over the user's whole history and each column narrows through its own `FILTER`; read it carefully before changing it.
- Keep the `::bigint` around the whole subtraction in that query. Without it sqlc types the result `int32`, which overflows past 2.1 tỷ đồng.
- Resolve percentages in Go, not the template: `balanceSummary` (built by `newBalanceSummary` in `balance_summary.go`). `html/template` cannot divide, and every percentage here (a month with no income, a month that overspent) needs a divide-by-zero guard.
- Only the balance is cumulative. The ratio bar and its caption still measure the displayed month against that month's own income.
- Both nav bars are in the DOM at once, so every mutation response returns `header_balance_oob`, which swaps two ids rather than relying on one selector. Wrapper spans carry `contents` so they leave no trace in the flex layout.

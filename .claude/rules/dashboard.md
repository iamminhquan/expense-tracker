---
paths:
  - "server/internal/api/dashboard_handlers.go"
  - "client/src/pages/DashboardPage.tsx"
---

# The dashboard

`server/internal/api/dashboard_handlers.go` builds every aggregate in Go; `client/src/pages/DashboardPage.tsx` renders them with Chart.js (`react-chartjs-2`).

## Server

- `buildPieData` feeds the doughnut with the top `pieTopN` = 6 categories plus a synthetic "Other" aggregate, so the chart never grows a tail of one-percent slivers. The real `other` default category is lifted out of the ranking and summed into that same aggregate rather than left to compete for one of the six spots -- see the function's own comment for why (the reserved `#A1A1AA` color would otherwise draw two identically-colored, identically-named slices).
- `buildBarSeries` feeds the `barMonths` = 4 month comparison and zero-pads any month the query returned no row for.
- Every number in the response is raw -- `previousTotalExpense`, a pie legend's `percent` as a plain `int`, an `amount` as a plain `int64` -- not a pre-formatted sentence or `template.JS`-wrapped JSON. There is no template here that can't itself divide or format a currency string; the client composes whatever sentence or chart call it needs, in whatever language it needs it in. Resolve category labels through `i18n` in Go regardless (the client has no copy of that table) -- see `categories.md`.
- `headerBalance`'s "always the real current month" rule lives in `.claude/rules/balance-widget.md`.

## Client

- `DashboardPage.tsx`'s own `comparison()` builds the "Last month X · up Y%" line from `previousTotalExpense`/`previousTotalIncome` + `hasPreviousMonthData` -- the client-side half of the split described above.
- The month picker is URL-driven (`useSearchParams`, not local `useState`) -- see `.claude/rules/req-value-objects.md` for why this matters (a bookmark or another page's link to a specific month has to actually show that month).
- `lib/charts.ts` registers Chart.js's elements once, imported for its side effect wherever a chart renders. Chart.js paints to a canvas, so a CSS variable changing under it changes nothing: the bar chart's colors (bars, legend text, axis ticks, grid) come from `hooks/useChartTheme.ts`, which reads the active palette off `:root` and re-reads it when the theme class flips or the OS color scheme does. The doughnut's slice colors are the categories' own, the same in both themes. Any new chart option that is a color has to come from that hook, not a literal.

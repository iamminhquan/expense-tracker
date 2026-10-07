---
paths:
  - "server/internal/api/dashboard_handlers.go"
  - "client/src/pages/dashboard/**"
  - "client/src/hooks/useThemeColors.ts"
---

# The dashboard

`server/internal/api/dashboard_handlers.go` builds every aggregate in Go; `client/src/pages/dashboard/` renders them: `KpiCards` (a bento: an accent hero for spent, a gold-tinted earned tile and a quiet net tile; the last two are slim rows on phones), `SpendingDoughnut` and `MonthlyBars` with Chart.js (`react-chartjs-2`).

## Server

- `buildPieData` feeds the doughnut with the top `pieTopN` = 6 categories plus a synthetic "Other" aggregate, so the chart never grows a tail of one-percent slivers. The real `other` default category is lifted out of the ranking and summed into that same aggregate rather than left to compete for one of the six spots -- see the function's own comment for why (the reserved `#A1A1AA` color would otherwise draw two identically-colored, identically-named slices).
- The legend's `percent` values always add up to exactly 100: `wholePercents` gives each slice its floor and hands the missing points to the largest remainders. Rounding each slice on its own let three equal thirds show as 33% + 33% + 33% = 99%; `TestDashboardPieLegendPercentsSumTo100` pins it.
- `buildBarSeries` feeds the `barMonths` = 4 month comparison and zero-pads any month the query returned no row for.
- Every number in the response is raw -- `previousTotalExpense`, a pie legend's `percent` as a plain `int`, an `amount` as a plain `int64` -- not a pre-formatted sentence or `template.JS`-wrapped JSON. There is no template here that can't itself divide or format a currency string; the client composes whatever sentence or chart call it needs, in whatever language it needs it in. Resolve category labels through `i18n` in Go regardless (the client has no copy of that table) -- see `categories.md`.
- `headerBalance`'s "always the real current month" rule lives in `.claude/rules/balance-widget.md`.

## Client

- `KpiCards.tsx`'s `Comparison` builds the "up Y% · Last month X" line from `previousTotalExpense`/`previousTotalIncome` + `hasPreviousMonthData` -- the client-side half of the split described above.
- The month picker is URL-driven (`useSearchParams`, not local `useState`) -- see `.claude/rules/req-value-objects.md` for why this matters (a bookmark or another page's link to a specific month has to actually show that month).
- `lib/charts.ts` registers Chart.js's elements once (and its default font), imported for its side effect wherever a chart renders. The legends are HTML next to the canvas, not Chart.js's own; each canvas has an `aria-label`, and the bar chart has a visually hidden table too.
- Chart colors come from `hooks/useThemeColors.ts`, which reads the `chart-expense`/`chart-income` CSS variables and the surface/border/ink ones and changes `key` on every theme switch. Both charts are keyed on it, so a theme switch rebuilds them, with animation off for the rebuild: `switched` turns true on the first switch in either direction and stays true. (Comparing against the key the chart mounted with missed a switch back to the starting theme, so dark → light replayed the draw-in animation.) Don't swap this for an in-place `updateMode="none"` update: Chart.js kept the bars' old resolved colors that way. Category slices use the stored hex colors, the same in both themes.

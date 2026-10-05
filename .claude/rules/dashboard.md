---
paths:
  - "server/internal/handlers/report_*.go"
  - "server/internal/web/templates/dashboard.html"
  - "server/internal/web/static/charts.js"
  - "server/internal/api/dashboard_handlers.go"
---

# The dashboard

`server/internal/handlers/report_handlers.go` builds everything in Go and hands the templates finished values.

## Rules

- `comparisonText` / `comparisonTextMobile` produce the "Last month X · down Y%" lines. There are two variants because the mobile cards share a row and have half the width.
- `buildPieData` feeds the doughnut with the top `pieTopN` = 6 categories plus a synthetic "Other" aggregate, so the chart never grows a tail of one-percent slivers.
- `buildBarSeries` feeds the `barMonths` = 4 month comparison and zero-pads any month the query returned no row for.
- Chart data crosses into JS as `template.JS`-wrapped JSON. Resolve category labels through `i18n` in Go here, not through a template func.

## `server/internal/api/dashboard_handlers.go` (Gin/JSON side, see `CLAUDE.md`)

- Same `pieTopN`/`barMonths`/`otherSlug` constants, same `buildPieData`/`buildBarSeries` logic, duplicated rather than shared (see `.claude/rules/req-value-objects.md` for why this migration duplicates rather than extracts).
- Deliberately drops `comparisonText`/`comparisonTextMobile` and `template.JS`-wrapped JSON: a JSON API ships raw numbers (`previousTotalExpense`, pie legend `percent` as a plain `int`, `amount` as a plain `int64`) and lets the client compose whatever sentence or chart library call it needs, in whatever language — there is no template here that cannot itself divide, so there is nothing forcing the percentage into a pre-formatted string server-side.
- Adds `headerBalance` (`handlers.currentHeaderBalance`'s JSON equivalent, same "always the real current month, never the month being browsed" rule) since a JSON client has no layout-level nav widget the server can render into — the client has to ask for this figure explicitly instead.

---
paths:
  - "server/internal/handlers/report_*.go"
  - "server/internal/web/templates/dashboard.html"
  - "server/internal/web/static/charts.js"
---

# The dashboard

`server/internal/handlers/report_handlers.go` builds everything in Go and hands the templates finished values.

## Rules

- `comparisonText` / `comparisonTextMobile` produce the "Last month X · down Y%" lines. There are two variants because the mobile cards share a row and have half the width.
- `buildPieData` feeds the doughnut with the top `pieTopN` = 6 categories plus a synthetic "Other" aggregate, so the chart never grows a tail of one-percent slivers.
- `buildBarSeries` feeds the `barMonths` = 4 month comparison and zero-pads any month the query returned no row for.
- Chart data crosses into JS as `template.JS`-wrapped JSON. Resolve category labels through `i18n` in Go here, not through a template func.

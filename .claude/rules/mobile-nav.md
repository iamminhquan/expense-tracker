---
paths:
  - "server/internal/web/templates/mobile_header.html"
  - "server/internal/web/static/categories.js"
  - "server/internal/web/static/app.js"
  - "server/internal/web/templates/nav.html"
---

# Mobile navigation

Blocks `nav_mobile_header` and `mobile_page_header` in `mobile_header.html`.

## Rules

- Below `md`, the nav collapses into a two-tier sticky header instead of the desktop `nav_desktop` bar: a slim top tier (logo + user menu) and a second tier with the page title, the month picker (dashboard/transactions) and an add button (transactions/categories).
- `mobile_page_header` is driven entirely by `.ActiveNav` and whatever `MonthLabel` / `CurrentMonthValue` / `AvailableMonths` the page's data already carries. Give it no page-specific params.
- Render it as the first child of the page's own swappable month/list section, never from the layout, so an htmx month switch carries the header along instead of leaving it behind.
- The mobile add-transaction sheet and the desktop quick-add form are both in the DOM. `handleCreateTransaction` reads `ui_source` to pick which fragment to re-render on a validation failure, and the Expense/Income toggle has two endpoints: `category_options` for the desktop `<select>`, `category_chips` for the sheet.
- Only category names go through `server/internal/i18n`. Every other string is written in English in the template or handler that shows it; there is no message catalog.

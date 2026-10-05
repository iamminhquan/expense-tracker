---
paths:
  - "server/internal/web/static/app.css"
  - "server/internal/web/static/tailwind-config.js"
  - "server/internal/web/static/charts.js"
  - "server/internal/handlers/view_layout_test.go"
  - "server/internal/web/static/app.js"
  - "server/internal/handlers/settings_theme.go"
  - "server/internal/handlers/category_handlers.go"
---

# Theming

## Colour

- All colour flows through CSS variables declared in `static/app.css` and referenced from `static/tailwind-config.js` as `rgb(var(--c-x) / <alpha-value>)`.
- The variables hold space-separated RGB channels, never hex. That is what keeps opacity modifiers like `bg-accent/10` working.
- Never hardcode a colour in a template (`text-[#6B6862]`, `style="background-color:#FEF7F5"`). Add or reuse a token.
- `view_layout_test.go` fails on a literal `rgba(` or `[#hex]` in a template, a utility class stranded outside a `class="..."` attribute, a `flex-1` form control with no width bound, and a bottom-sheet grab handle that has drifted from the `[data-sheet-handle]` selector `app.js` looks for.

## Light, dark, auto

- The dark palette is declared twice, under `@media (prefers-color-scheme: dark) :root:not(.light)` and under `:root.dark`. Keep both in step. The three preferences (`auto` / `light` / `dark`) then resolve in CSS alone, with no load-time JavaScript and no flash.
- The preference lives in `users.theme` (CHECK-constrained, mirrored by `settings_theme.go`'s `validTheme`) and is rendered onto `<html class="...">`. `renderNamed` defaults it to `auto` for pre-auth pages, which have no user.
- Chart.js cannot read CSS variables. `static/charts.js` resolves them via `chartColor()` at construction and rebuilds both charts on the `themechange` event, which the switch (and an OS flip while on `auto`) dispatches.

## Category palette

- The set is fixed: 8 user-selectable swatches in `categorySwatches` (`category_handlers.go`), plus the reserved `#A1A1AA` grey for the "Other" default and the chart's synthetic aggregate.
- It is enforced twice: `isValidSwatch` in Go and `categories_color_check` in migration 000006.

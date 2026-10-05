---
paths:
  - "client/src/index.css"
  - "client/src/lib/theme/ThemeContext.tsx"
  - "client/src/lib/charts.ts"
  - "server/internal/api/category_handlers.go"
  - "client/src/pages/CategoriesPage.tsx"
---

# Theming

## Colour

- All colour flows through CSS custom properties declared in `client/src/index.css`'s `:root` and mapped into Tailwind's color namespace via its `@theme` block -- `index.css`'s own comments explain the port from the old `app.css`/`tailwind-config.js` pair, including why Tailwind v4 no longer needs the `<alpha-value>` placeholder trick v3's Play CDN config required (opacity modifiers like `bg-accent/10` work automatically via `color-mix()` now, whatever format the underlying value is in).
- The variables hold space-separated RGB channels, never hex, the same convention the deleted `app.css` used.
- Never hardcode a colour in a component (`text-[#6B6862]`, `style={{backgroundColor: '#FEF7F5'}}`). Add or reuse a token in `index.css`'s `@theme` block instead.
- There is no automated layout-invariant test here the way `view_layout_test.go` was for the old templates (it was deleted with `internal/handlers`). Nothing currently catches a stray hardcoded color or a Tailwind utility class used outside a component's className -- a gap, not a deliberate choice, if you're looking for something to add.

## Light, dark, auto

- The dark palette is declared twice in `index.css`, under `@media (prefers-color-scheme: dark) :root:not(.light)` and under `:root.dark`. Keep both in step. The three preferences (`auto` / `light` / `dark`) then resolve in CSS alone once the class is set, with no load-time flash.
- `ThemeContext.tsx`'s `applyTheme` sets that class on `<html>`, mirroring the deleted `static/app.js`'s `applyTheme()`. The preference lives in `users.theme` (CHECK-constrained, mirrored by `server/internal/api/settings_handlers.go`'s `validTheme`) and is carried on `/api/me`/`/api/refresh`'s response (`userDTO.Theme` -- see that struct's own comment for why it rides there and not only on `/api/settings`) rather than being rendered server-side onto an HTML root.
- `ThemeProvider` must be nested inside `AuthProvider`: there is no user, and so no saved preference, until auth resolves. Pre-auth pages default to `auto`.
- **Known gap:** Chart.js's colors in `DashboardPage.tsx` are hardcoded to the light palette's RGB values rather than resolved from the active theme and rebuilt on a theme change, the way the deleted `static/charts.js`'s `chartColor()` + `themechange` listener did. See `.claude/rules/dashboard.md`.

## Category palette

- The set is fixed: 8 user-selectable swatches, duplicated as `categorySwatches` in `server/internal/api/category_handlers.go` (server-side validation) and `SWATCHES` in `client/src/pages/CategoriesPage.tsx` (the picker UI) -- plus the reserved `#A1A1AA` grey for the "Other" default and the dashboard chart's synthetic aggregate slice, which is server-only and never offered as a user choice.
- Keep both lists in the same order and the same values. `isValidSwatch` (Go) and migration 000006's `categories_color_check` constraint are what actually enforce the set; the client list exists only to draw the picker and would just produce a rejected request if it drifted.

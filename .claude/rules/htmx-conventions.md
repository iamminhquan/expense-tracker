---
paths:
  - "server/internal/handlers/*.go"
  - "server/internal/auth/middleware.go"
  - "server/internal/web/templates/*.html"
  - "server/internal/web/static/app.js"
---

# htmx conventions

## Rules

- Mutation handlers (add/edit/delete transaction or category) return HTML fragments swapped into the DOM, never JSON.
- Each one also returns its refreshed out-of-band companions: `header_balance_oob` always, plus `totals_oob` (count, empty state, pager) on the transactions page.
- Never 3xx-redirect an htmx XHR on session expiry: htmx would swap the full login page into whatever partial was targeted. `redirectToLogin` in `server/internal/auth/middleware.go` sets `HX-Redirect` when `HX-Request: true`, which htmx turns into a real top-level navigation. Login/register success in `auth_handlers.go` uses the same pattern.
- The settings forms are the exception: plain `hx-boost`ed POSTs that redirect with `?saved=` on success, so a reload or back button doesn't re-submit.

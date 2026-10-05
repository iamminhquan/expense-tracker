---
paths:
  - "internal/csrf/*.go"
  - "internal/web/static/app.js"
  - "internal/web/templates/layout.html"
---

# CSRF

## Rules

- `internal/csrf/csrf.go` is a stateless double-submit-cookie pattern. There is no server-side token storage; every request gets a `csrf_token` cookie.
- A mutating request must echo the cookie back, either:
  - as the `X-CSRF-Token` header for htmx requests (the `htmx:configRequest` listener in `static/app.js` copies the `<meta name="csrf-token">` value into it), or
  - as a hidden `csrf_token` field for plain `<form method="POST">` submissions, e.g. logout.
- Every route goes through `csrf.Middleware`; there is no exempt route.

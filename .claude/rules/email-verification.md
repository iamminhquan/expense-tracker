---
paths:
  - "server/internal/auth/email_verification.go"
  - "server/internal/handlers/auth_email_verification.go"
  - "server/internal/web/templates/layout.html"
  - "server/internal/web/templates/verify_email.html"
  - "server/internal/mailer/**"
  - "server/internal/handlers/settings_handlers.go"
  - "server/internal/api/password_reset_handlers.go"
  - "server/internal/api/settings_handlers.go"
---

# Email verification

Code: `server/internal/auth/email_verification.go` and `server/internal/handlers/auth_email_verification.go`.

## Rules

- It mirrors forgot-password against its own `email_verification_tokens` table, with a 24h TTL rather than the reset link's 1h. Confirming an address is never the urgency of a locked-out owner.
- One token type serves both entry points a link can prove, a fresh signup and a settings email change. Both ask "can this account be reached here", and `ApplyVerifiedEmail` answers the same way each time: copy the proven address onto `users.email` and flip `email_verified`.
- `updateEmailHandler` never writes `users.email` directly. It stages the change on `pending_email`, and only `ApplyVerifiedEmail` promotes it once the link sent to the *new* address is visited.
- The pre-check that rejects an already-registered address excludes the caller's own row, so resubmitting the current address is a no-op rather than a false collision.
- Never block an unverified account from anything. Every authenticated page shows a small reminder banner (in `layout.html`, gated on `EmailVerified` from `authPageView`) until the address is confirmed. Its resend link reissues whichever address is unconfirmed: `pending_email` if a change is in flight, otherwise `email`.
- Migration 000013 grandfathers in every account that existed before the check as verified.

## Why

`users.email` is also the login identity and the address a forgot-password link goes to, so applying a change immediately would let one typo cost the owner both with no way back in. Holding it means a typo just leaves `pending_email` unconfirmed: the owner keeps logging in on the address that was always correct and can resubmit.

## Gin/JSON side (`internal/api`, see `CLAUDE.md`)

- `password_reset_handlers.go` holds forgot/reset-password (`POST /api/forgot-password`, `GET`+`POST /api/reset-password`) and email verification (`POST /api/verify-email`) -- all four duplicate their `handlers/auth_password_reset.go` / `auth_email_verification.go` counterparts' logic exactly, same token tables, same TTLs, same enumeration-safe forgot-password response.
- `verify-email` is `POST` with the token in a JSON body, not a `GET` with it in the query string: the link a verification email points at opens the React app's own `/verify-email` route (`client/`, Phase 2), which reads `?token=` itself and POSTs it to this endpoint -- the API never renders the landing page the way `handlers.verifyEmailPage` does.
- `resetPasswordHandler` signs the visitor in on success (an `authResponse`, same as register/login), matching `handlers.resetPasswordPage`'s `startSession` call -- there is no current session to spare on this path, unlike `updatePasswordHandler`'s.

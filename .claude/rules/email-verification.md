---
paths:
  - "server/internal/auth/email_verification.go"
  - "server/internal/api/password_reset_handlers.go"
  - "server/internal/api/settings_handlers.go"
  - "server/internal/mailer/**"
  - "client/src/components/layout/VerifyEmailBanner.tsx"
---

# Email verification

Code: `server/internal/auth/email_verification.go` and `server/internal/api/password_reset_handlers.go` (`POST /api/verify-email`).

## Rules

- It mirrors forgot-password against its own `email_verification_tokens` table, with a 24h TTL rather than the reset link's 1h. Confirming an address is never the urgency of a locked-out owner.
- One token type serves both entry points a link can prove, a fresh signup and a settings email change. Both ask "can this account be reached here", and `ApplyVerifiedEmail` answers the same way each time: copy the proven address onto `users.email` and flip `email_verified`.
- `updateEmailHandler` (`internal/api/settings_handlers.go`) never writes `users.email` directly. It stages the change on `pending_email`, and only `ApplyVerifiedEmail` promotes it once the link sent to the *new* address is visited.
- The pre-check that rejects an already-registered address excludes the caller's own row, so resubmitting the current address is a no-op rather than a false collision.
- `verifyEmailHandler` takes its token in a `POST` body, not a `GET` query string: the link an email points at opens the client's own `/verify-email` route, which reads `?token=` itself and POSTs it here. It's unauthenticated on purpose, same reasoning as forgot/reset-password -- the browser opening the link is often not the one the visitor is signed in on, and the token itself is what proves the request is legitimate.
- `resendVerificationHandler` reissues whichever address is unconfirmed: `pending_email` if a change is in flight, otherwise `email`.
- Migration 000013 grandfathers in every account that existed before the check as verified.
- `userDTO.emailVerified` feeds `client/`'s `VerifyEmailBanner`, a reminder strip with a resend button. It is only a reminder: never block an unverified account from anything. A pending email *change* does not show it, since `users.email` is still the verified old address; the Settings email card shows that instead.

## Why

`users.email` is also the login identity and the address a forgot-password link goes to, so applying a change immediately would let one typo cost the owner both with no way back in. Holding it means a typo just leaves `pending_email` unconfirmed: the owner keeps logging in on the address that was always correct and can resubmit.

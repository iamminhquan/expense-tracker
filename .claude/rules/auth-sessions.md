---
paths:
  - "server/internal/auth/*.go"
  - "server/internal/handlers/settings_handlers.go"
  - "server/internal/format/device.go"
  - "server/internal/handlers/auth_*.go"
  - "server/internal/api/auth_handlers.go"
  - "server/internal/api/settings_handlers.go"
  - "server/internal/api/middleware.go"
---

# Auth: sessions, lockout, active sessions

## Sessions

- `server/internal/auth/session.go` issues and validates sessions against the `sessions` table. Its functions, like the reset- and verification-token files beside it, take `*sqlcgen.Queries` as an argument rather than a type wrapping one, so a caller inside a transaction can pass `Queries.WithTx(tx)`.
- `password.go` handles bcrypt hashing and verification.
- Sessions last 7 days. A password change deletes every *other* session for that user and keeps the current one.

## Lockout

- `lockout.go` holds the throttle's numbers (`MaxLoginAttempts` = 5, `LockoutWindow` = 15 minutes): 5 consecutive wrong passwords lock an account for 15 minutes. The SQL that stamps the lock and the message that explains it must agree, so change the numbers there only.
- The state is two columns on `users` (`failed_login_attempts`, `locked_until`), not a table. Only real accounts are counted and a lapsed lock needs no sweeping.
- `RecordFailedLogin` counts and locks in one UPDATE. Don't do a read-modify-write in Go: a parallel flood would spend far more than 5 guesses.
- Check the lock *before* the password, so guessing at a locked account can't extend the window. A completed password reset clears the lock; that is the only way out besides waiting.

## Active sessions (`/settings`)

- `created_at` and `user_agent` (migration 000012) exist only for this list. `user_agent` is nullable because sessions older than the migration have none.
- `format.DeviceLabel` (`server/internal/format/device.go`) turns the raw UA into "Chrome on Windows" by matching a few common substrings. Fall back to the raw string rather than guess wrong.
- Signing out one device goes through `DeleteSessionForUser`, scoped to `user_id` as well as `id`, so a typed `session_id` can never reach a row it doesn't own.
- "Log out everywhere else" calls `DeleteOtherSessionsForUser`, the same call a password change uses, here as a deliberate action.
- The current session never gets its own revoke button, so a click can't log the viewer out of the page they are on.

## Gin/JSON side (`internal/api`, see `CLAUDE.md`)

- There is no second session mechanism. `internal/api`'s refresh token *is* this same `sessions` row/cookie, just read through `internal/auth`'s existing functions instead of through `internal/auth.RequireAuth`'s cookie-only middleware -- see `internal/auth/jwt.go`'s doc comment for the full reasoning (a stateless access token on top, 15 min TTL, issued/validated by `internal/api.RequireAuth`).
- Everything above this section (lockout, 7-day TTL, revoke-one/revoke-others, device labels) applies identically to the Gin side; nothing here is reimplemented differently, only the request plumbing (JSON body/Authorization header instead of a form POST/cookie) changed.
- `internal/api.UserID(c)` is the Gin-context equivalent of `auth.UserIDFromContext(ctx)`.

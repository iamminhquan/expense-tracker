---
paths:
  - "server/internal/auth/*.go"
  - "server/internal/format/device.go"
  - "server/internal/api/auth_handlers.go"
  - "server/internal/api/settings_handlers.go"
  - "server/internal/api/middleware.go"
---

# Auth: sessions, lockout, active sessions

## Sessions and access tokens

- `server/internal/auth/session.go` issues and validates a random opaque token against the `sessions` table. Its functions, like the reset- and verification-token files beside it, take `*sqlcgen.Queries` as an argument rather than a type wrapping one, so a caller inside a transaction can pass `Queries.WithTx(tx)`.
- That same token is the refresh token the API sends as an httpOnly cookie (`Path=/api`) -- there is no second session mechanism. `internal/auth/jwt.go`'s doc comment has the full reasoning: a stateless access token (15 min TTL, issued/validated by `internal/api.RequireAuth`) sits on top, and the refresh token is what's actually revocable (logout, "log out everywhere else", a password change).
- `password.go` handles bcrypt hashing and verification.
- Every `POST /api/v1/refresh` replaces the session's token (`auth.RefreshSession`, migration 000020); `server.md`'s Authentication Model has the grace-period and reuse rules. When touching refresh, logout or revoke queries, remember the row can be addressed by `id` (current token) or `previous_id` (the one just replaced).
- Sessions last 7 days, counted from login: rotation does not extend them. A password change deletes every *other* session for that user and keeps the current one.
- `internal/api.UserID(c)` reads the authenticated user's ID an access token named; it's the Gin-context equivalent of a context value.
- The refresh-token cookie's `SameSite` must track `SecureCookies` (`refreshCookieSameSite` in `auth_handlers.go`), never a hardcoded `None` -- see `.claude/rules/json-api-conventions.md` for why a hardcoded `None` silently broke every hard page reload in local dev (Chrome drops a `SameSite=None` cookie outright without `Secure`, and `SecureCookies` is false there).

## Lockout

- `lockout.go` holds the throttle's numbers (`MaxLoginAttempts` = 5, `LockoutWindow` = 15 minutes): 5 consecutive wrong passwords lock an account for 15 minutes. The SQL that stamps the lock and the message that explains it must agree, so change the numbers there only.
- The state is two columns on `users` (`failed_login_attempts`, `locked_until`), not a table. Only real accounts are counted and a lapsed lock needs no sweeping.
- `RecordFailedLogin` counts and locks in one UPDATE. Don't do a read-modify-write in Go: a parallel flood would spend far more than 5 guesses.
- Check the lock *before* the password, so guessing at a locked account can't extend the window. A completed password reset clears the lock; that is the only way out besides waiting.

## Active sessions (`GET /api/v1/settings`)

- `created_at` and `user_agent` (migration 000012) exist only for this list. `user_agent` is nullable because sessions older than the migration have none.
- `format.DeviceLabel` (`server/internal/format/device.go`) turns the raw UA into "Chrome on Windows" by matching a few common substrings. Fall back to the raw string rather than guess wrong.
- The list's `id` is `sessions.public_id` (migration 000019), never `sessions.id`: that one is the refresh token, and sending it to the page would put every active refresh token, the viewer's own included, within reach of page scripts and make the cookie's httpOnly flag moot. Never put `sessions.id` in a response body.
- Signing out one device goes through `DeleteSessionForUser`, keyed on `public_id` and scoped to `user_id`, so an id a client sends can never reach a row it doesn't own. A malformed id is a 400.
- "Log out everywhere else" calls `DeleteOtherSessionsForUser`, the same call a password change uses, here as a deliberate action.
- The server marks "the current session" (`isCurrent`) by comparing a row's `id` against the refresh-token cookie the request itself carried, and the client never offers that one a revoke button -- a click can't sign the viewer out of the device they're using it from.

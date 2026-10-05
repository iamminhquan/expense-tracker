// The access token lives here -- a plain module-level variable, not React
// state -- so client.ts (a bag of plain functions, not hooks) can read and
// refresh it without importing AuthContext and creating a cycle.
// AuthProvider is the only thing that calls setAccessToken; everything else
// only reads through client.ts's request() function.
//
// This is deliberately *not* persisted (no localStorage/sessionStorage):
// the migration plan's locked JWT-storage decision keeps the access token
// in memory only, so it never survives a reload -- client.ts's silent
// refresh (via the httpOnly refresh-token cookie) is what AuthProvider
// calls on mount to recover it instead.
let accessToken: string | null = null

export function getAccessToken(): string | null {
  return accessToken
}

export function setAccessToken(token: string | null): void {
  accessToken = token
}

// unauthorizedHandler is AuthProvider's "the refresh token is gone too, the
// visitor is signed out" callback. A setter rather than an event emitter
// because there is ever only one AuthProvider mounted.
let unauthorizedHandler: (() => void) | null = null

export function setUnauthorizedHandler(fn: (() => void) | null): void {
  unauthorizedHandler = fn
}

export function notifyUnauthorized(): void {
  unauthorizedHandler?.()
}

/*
 * In memory only, never localStorage/sessionStorage: a reload loses it and
 * AuthProvider recovers it with a silent refresh. A module variable rather
 * than React state, so client.ts can read it without importing AuthContext.
 */
let accessToken: string | null = null

export function getAccessToken(): string | null {
  return accessToken
}

export function setAccessToken(token: string | null): void {
  accessToken = token
}

// Set by AuthProvider; called once the refresh token is gone too.
let unauthorizedHandler: (() => void) | null = null

export function setUnauthorizedHandler(fn: (() => void) | null): void {
  unauthorizedHandler = fn
}

export function notifyUnauthorized(): void {
  unauthorizedHandler?.()
}

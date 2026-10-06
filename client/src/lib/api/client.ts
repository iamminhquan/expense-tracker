import { getAccessToken, notifyUnauthorized, setAccessToken } from './tokenStore'

// Empty in dev (Vite proxies /api); the API's origin in production.
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? ''

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
}

// Unwraps the envelope every /api/v1 JSON response carries, throwing ApiError on failure.
export async function readApiResponse<T>(res: Response): Promise<T> {
  let body: ApiResponse<T> | undefined
  try {
    body = (await res.json()) as ApiResponse<T>
  } catch {
    // Not JSON (a proxy's error page, say): fall back to the status text.
  }
  if (!res.ok || !body?.success) {
    throw new ApiError(res.status, body?.message || res.statusText || `request failed with status ${res.status}`)
  }
  return body.data
}

interface RequestOptions {
  method?: string
  body?: unknown
  /** Public endpoints: no token, and no 401 retry (it would recurse into refresh). */
  skipAuth?: boolean
}

function send(path: string, init: RequestInit, withAuth: boolean): Promise<Response> {
  const headers = new Headers(init.headers)
  if (withAuth) {
    const token = getAccessToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }
  return fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    // Needed for the refresh-token cookie, which is cross-origin in production.
    credentials: 'include',
  })
}

/*
 * Concurrent 401s share one refresh call. Separate refreshes would race
 * and invalidate each other's new token.
 */
let refreshInFlight: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = (async () => {
    try {
      const res = await send('/api/v1/refresh', { method: 'POST' }, false)
      if (!res.ok) return null
      const data = await readApiResponse<{ accessToken: string }>(res)
      setAccessToken(data.accessToken)
      return data.accessToken
    } catch {
      return null
    }
  })()
  try {
    return await refreshInFlight
  } finally {
    refreshInFlight = null
  }
}

/*
 * For requests that can't go through api.* (multipart uploads, file downloads).
 * Sends the access token; on a 401 refreshes it once and sends the same request
 * again, so `init.body` must be reusable (a string, FormData, not a stream).
 * The caller reads the Response itself.
 */
export async function authedFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const res = await send(path, init, true)
  if (res.status !== 401) return res

  const newToken = await refreshAccessToken()
  if (newToken) return send(path, init, true)
  setAccessToken(null)
  notifyUnauthorized()
  return res
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const init: RequestInit = {
    method: options.method ?? 'GET',
    headers: options.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  }
  const res = options.skipAuth ? await send(path, init, false) : await authedFetch(path, init)
  return readApiResponse<T>(res)
}

type Opts = Omit<RequestOptions, 'method' | 'body'>

export const api = {
  get: <T,>(path: string, opts?: Opts) => request<T>(path, { ...opts, method: 'GET' }),
  post: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'POST', body }),
  patch: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'PATCH', body }),
  put: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'PUT', body }),
  delete: <T,>(path: string, opts?: Opts) => request<T>(path, { ...opts, method: 'DELETE' }),
}

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

interface RequestOptions {
  method?: string
  body?: unknown
  /** Public endpoints: no token, and no 401 retry (it would recurse into refresh). */
  skipAuth?: boolean
}

async function doFetch(path: string, options: RequestOptions): Promise<Response> {
  const headers: Record<string, string> = {}
  if (options.body !== undefined) headers['Content-Type'] = 'application/json'
  if (!options.skipAuth) {
    const token = getAccessToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }
  return fetch(`${API_BASE}${path}`, {
    method: options.method ?? 'GET',
    headers,
    // Needed for the refresh-token cookie, which is cross-origin in production.
    credentials: 'include',
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
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
      const res = await doFetch('/api/v1/refresh', { method: 'POST', skipAuth: true })
      if (!res.ok) return null
      const data = (await res.json()) as { accessToken: string }
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

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  let res = await doFetch(path, options)

  if (res.status === 401 && !options.skipAuth) {
    const newToken = await refreshAccessToken()
    if (newToken) {
      res = await doFetch(path, options)
    } else {
      setAccessToken(null)
      notifyUnauthorized()
    }
  }

  if (!res.ok) {
    let message = res.statusText || `request failed with status ${res.status}`
    try {
      const data = (await res.json()) as { error?: string }
      if (data.error) message = data.error
    } catch {
      // Not JSON: keep the statusText fallback.
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

type Opts = Omit<RequestOptions, 'method' | 'body'>

export const api = {
  get: <T,>(path: string, opts?: Opts) => request<T>(path, { ...opts, method: 'GET' }),
  post: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'POST', body }),
  patch: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'PATCH', body }),
  put: <T,>(path: string, body?: unknown, opts?: Opts) => request<T>(path, { ...opts, method: 'PUT', body }),
  delete: <T,>(path: string, opts?: Opts) => request<T>(path, { ...opts, method: 'DELETE' }),
}

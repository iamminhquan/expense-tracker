import { getAccessToken, notifyUnauthorized, setAccessToken } from './tokenStore'

// Relative in dev (Vite's proxy in vite.config.ts forwards /api to
// localhost:8080, so the browser sees one origin); an absolute URL
// (VITE_API_BASE_URL) in production, where client/ (Vercel) and server/
// (Render) are genuinely cross-origin.
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
  // skipAuth is for the handful of endpoints that are public (login,
  // register, forgot/reset-password, refresh itself) -- a request to one
  // of these never attaches a stale Authorization header and never
  // triggers the 401-retry-via-refresh loop below, which would otherwise
  // recurse into refresh's own request.
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
    // Required for the refresh-token cookie to be sent/received at all,
    // same-origin in dev or genuinely cross-origin in production -- see
    // internal/api/router.go's CORS middleware on the server side.
    credentials: 'include',
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  })
}

// refreshInFlight collapses concurrent 401s into one /api/refresh call:
// several requests failing at once (e.g. a page that fires off three
// queries on load with an access token that just expired) would otherwise
// each race their own refresh and invalidate each other's new token.
let refreshInFlight: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = (async () => {
    try {
      const res = await doFetch('/api/refresh', { method: 'POST', skipAuth: true })
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
      // Body wasn't JSON (a network-level error page, an empty body) --
      // keep the statusText fallback above.
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

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, readApiResponse } from './client'
import { getAccessToken, setAccessToken, setUnauthorizedHandler } from './tokenStore'

function envelope(data: unknown, status = 200, success = true, message = 'ok'): Response {
  return new Response(JSON.stringify({ success, message, data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function failure(status: number, message: string): Response {
  return envelope(null, status, false, message)
}

const urlOf = (input: RequestInfo | URL) => (typeof input === 'string' ? input : input.toString())

const header = (init: RequestInit | undefined, name: string) => new Headers(init?.headers).get(name)

beforeEach(() => {
  setAccessToken(null)
  setUnauthorizedHandler(null)
})

describe('readApiResponse', () => {
  it('returns the envelope data on success', async () => {
    await expect(readApiResponse<{ id: number }>(envelope({ id: 7 }))).resolves.toEqual({ id: 7 })
  })

  it('throws an ApiError carrying the server message and status', async () => {
    const err = await readApiResponse(failure(409, 'that email is already registered')).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 409, message: 'that email is already registered' })
  })

  it('falls back to the status text when the body is not JSON', async () => {
    const res = new Response('<html>Bad gateway</html>', { status: 502, statusText: 'Bad Gateway' })
    await expect(readApiResponse(res)).rejects.toMatchObject({ status: 502, message: 'Bad Gateway' })
  })

  it('treats success: false on a 200 as a failure', async () => {
    await expect(readApiResponse(envelope(null, 200, false, 'nope'))).rejects.toMatchObject({ message: 'nope' })
  })
})

describe('api requests', () => {
  it('sends the access token and credentials, and JSON bodies with a content type', async () => {
    setAccessToken('tok-1')
    const fetchMock = vi.fn().mockResolvedValue(envelope({ ok: true }))
    vi.stubGlobal('fetch', fetchMock)

    await api.post('/api/v1/categories', { name: 'Food' })

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/categories')
    expect(init.method).toBe('POST')
    expect(init.credentials).toBe('include')
    expect(init.body).toBe(JSON.stringify({ name: 'Food' }))
    expect(header(init, 'Authorization')).toBe('Bearer tok-1')
    expect(header(init, 'Content-Type')).toBe('application/json')
  })

  it('on a 401 refreshes once, then retries with the new token', async () => {
    setAccessToken('expired')
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (urlOf(input) === '/api/v1/refresh') return envelope({ accessToken: 'fresh' })
      return header(init, 'Authorization') === 'Bearer fresh' ? envelope({ n: 1 }) : failure(401, 'expired')
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/api/v1/dashboard')).resolves.toEqual({ n: 1 })
    expect(getAccessToken()).toBe('fresh')
    expect(fetchMock.mock.calls.map(([u]) => urlOf(u as string))).toEqual([
      '/api/v1/dashboard',
      '/api/v1/refresh',
      '/api/v1/dashboard',
    ])
  })

  // Rotating refresh tokens make this load-bearing: a second refresh with the
  // same cookie would race the first.
  it('shares one refresh between requests that fail with 401 together', async () => {
    setAccessToken('expired')
    let refreshes = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (urlOf(input) === '/api/v1/refresh') {
          refreshes++
          await new Promise((r) => setTimeout(r, 10))
          return envelope({ accessToken: 'fresh' })
        }
        return header(init, 'Authorization') === 'Bearer fresh' ? envelope({ path: urlOf(input) }) : failure(401, 'expired')
      }),
    )

    const results = await Promise.all([api.get('/api/v1/a'), api.get('/api/v1/b'), api.get('/api/v1/c')])

    expect(refreshes).toBe(1)
    expect(results).toEqual([{ path: '/api/v1/a' }, { path: '/api/v1/b' }, { path: '/api/v1/c' }])
  })

  it('refreshes again for a later 401, not remembering the finished one', async () => {
    setAccessToken('expired')
    let refreshes = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (urlOf(input) === '/api/v1/refresh') {
          refreshes++
          return envelope({ accessToken: `fresh-${refreshes}` })
        }
        return header(init, 'Authorization') === `Bearer fresh-${refreshes}` && refreshes > 0
          ? envelope(null)
          : failure(401, 'expired')
      }),
    )

    await api.get('/api/v1/a')
    setAccessToken('expired-again')
    await api.get('/api/v1/b')

    expect(refreshes).toBe(2)
  })

  it('clears the token and signs out when the refresh fails too', async () => {
    setAccessToken('expired')
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) =>
        urlOf(input) === '/api/v1/refresh' ? failure(401, 'refresh token expired') : failure(401, 'expired'),
      ),
    )

    await expect(api.get('/api/v1/dashboard')).rejects.toMatchObject({ status: 401 })
    expect(getAccessToken()).toBeNull()
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('does not refresh for a public endpoint, whose 401 is just a wrong password', async () => {
    const fetchMock = vi.fn().mockResolvedValue(failure(401, 'invalid email or password'))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.post('/api/v1/login', { email: 'a', password: 'b' }, { skipAuth: true })).rejects.toMatchObject({
      message: 'invalid email or password',
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('does not send a token to a public endpoint', async () => {
    setAccessToken('tok')
    const fetchMock = vi.fn().mockResolvedValue(envelope(null))
    vi.stubGlobal('fetch', fetchMock)

    await api.post('/api/v1/forgot-password', { email: 'a@b.c' }, { skipAuth: true })

    expect(header(fetchMock.mock.calls[0][1] as RequestInit, 'Authorization')).toBeNull()
  })
})

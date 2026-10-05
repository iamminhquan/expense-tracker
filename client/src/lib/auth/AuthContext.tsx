import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import * as authApi from '../api/auth'
import { setAccessToken, setUnauthorizedHandler } from '../api/tokenStore'
import type { AuthResponse, User } from '../api/types'

type Status = 'loading' | 'authenticated' | 'unauthenticated'

interface AuthContextValue {
  user: User | null
  status: Status
  login: (email: string, password: string) => Promise<void>
  register: (input: authApi.RegisterInput) => Promise<void>
  logout: () => Promise<void>
  // setSession adopts an AuthResponse a page got from some other
  // endpoint that also signs the visitor in -- today just
  // resetPasswordHandler's response -- without duplicating login's
  // request. Calling only tokenStore.setAccessToken from such a page
  // would leave `user`/`status` here stale, which is what
  // ProtectedRoute actually renders against.
  setSession: (res: AuthResponse) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

// AuthProvider owns the access token's lifecycle. The token itself lives
// in tokenStore.ts (a plain module variable client.ts reads directly, not
// React state) -- see that file's comment for why -- but this is the only
// place that ever calls setAccessToken, and `user`/`status` here are what
// the rest of the app actually renders against.
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<Status>('loading')

  // On mount, try a silent refresh: a reload has lost the in-memory access
  // token by design, but the httpOnly refresh-token cookie survives it, so
  // this recovers a signed-in visitor's session without asking them to log
  // in again every time they reload the page.
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch(
          `${(import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''}/api/v1/refresh`,
          { method: 'POST', credentials: 'include' },
        )
        if (!res.ok) throw new Error('no refresh token')
        const data = (await res.json()) as { accessToken: string; user: User }
        if (cancelled) return
        setAccessToken(data.accessToken)
        setUser(data.user)
        setStatus('authenticated')
      } catch {
        if (cancelled) return
        setAccessToken(null)
        setUser(null)
        setStatus('unauthenticated')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  // client.ts calls this when a request's 401 survives a refresh attempt
  // -- the refresh token itself is gone (expired, revoked from Settings,
  // or the account was deleted) -- so the app has to drop back to signed
  // out rather than keep rendering protected pages against a dead session.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      setUser(null)
      setStatus('unauthenticated')
    })
    return () => setUnauthorizedHandler(null)
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login({ email, password })
    setAccessToken(res.accessToken)
    setUser(res.user)
    setStatus('authenticated')
  }, [])

  const register = useCallback(async (input: authApi.RegisterInput) => {
    const res = await authApi.register(input)
    setAccessToken(res.accessToken)
    setUser(res.user)
    setStatus('authenticated')
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      setAccessToken(null)
      setUser(null)
      setStatus('unauthenticated')
    }
  }, [])

  const setSession = useCallback((res: AuthResponse) => {
    setAccessToken(res.accessToken)
    setUser(res.user)
    setStatus('authenticated')
  }, [])

  const value = useMemo(
    () => ({ user, status, login, register, logout, setSession }),
    [user, status, login, register, logout, setSession],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be called within an AuthProvider')
  return ctx
}

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import * as authApi from '../api/auth'
import { readApiResponse } from '../api/client'
import { setAccessToken, setUnauthorizedHandler } from '../api/tokenStore'
import type { AuthResponse, User } from '../api/types'

type Status = 'loading' | 'authenticated' | 'unauthenticated'

interface AuthContextValue {
  user: User | null
  status: Status
  login: (email: string, password: string) => Promise<void>
  register: (input: authApi.RegisterInput) => Promise<void>
  logout: () => Promise<void>
  /** Adopts a sign-in response from another endpoint, e.g. reset-password. */
  setSession: (res: AuthResponse) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<Status>('loading')

  // A reload loses the in-memory token; the refresh-token cookie restores the session.
  useEffect(() => {
    let cancelled = false
      ; (async () => {
        try {
          const res = await fetch(
            `${(import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''}/api/v1/refresh`,
            { method: 'POST', credentials: 'include' },
          )
          const data = await readApiResponse<{ accessToken: string; user: User }>(res)
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

  // The refresh token is gone too (expired, revoked, account deleted): sign out.
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

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { useAuth } from '../auth/AuthContext'
import * as settingsApi from '../api/settings'
import type { Theme } from '../api/types'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

// Mirrors server/internal/web/static/app.js's applyTheme(): the class on
// <html> is the only thing that matters for which palette renders --
// "auto" leaves CSS's prefers-color-scheme media query (index.css) to
// pick light or dark, so the client never has to read
// window.matchMedia itself.
function applyTheme(theme: Theme) {
  document.documentElement.classList.remove('light', 'dark')
  if (theme !== 'auto') {
    document.documentElement.classList.add(theme)
  }
}

// ThemeProvider must be nested inside AuthProvider: the signed-in user's
// preference (user.theme, carried on /api/me and /api/refresh's response
// -- see server/internal/api/auth_handlers.go's userDTO) is the source of
// truth once authenticated. Pre-auth (login/register pages) there is no
// user to load one from, so it defaults to "auto" the same way
// handlers.defaultTheme does.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [theme, setThemeState] = useState<Theme>('auto')

  useEffect(() => {
    const next = user?.theme ?? 'auto'
    setThemeState(next)
    applyTheme(next)
  }, [user])

  const setTheme = (next: Theme) => {
    // Applied immediately, before the request resolves -- the switch has
    // already recolored the page by the time anything could fail, same
    // as handlers/settings_theme.go's comment on the HTML side, so there
    // is nothing to roll back to on a failed PUT besides leaving the
    // local choice in place.
    setThemeState(next)
    applyTheme(next)
    void settingsApi.updateTheme(next).catch(() => {
      // Best-effort: a failed save leaves the local preference applied
      // for this session but unsaved server-side, matching the HTML
      // side's own fire-and-forget PUT (it answers 204 with nothing to
      // swap back in on success, and has no failure UI either).
    })
  }

  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be called within a ThemeProvider')
  return ctx
}

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { useAuth } from '../auth/AuthContext'
import * as settingsApi from '../api/settings'
import type { Theme } from '../api/types'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

// "auto" sets no class and leaves it to index.css's prefers-color-scheme query.
function applyTheme(theme: Theme) {
  document.documentElement.classList.remove('light', 'dark')
  if (theme !== 'auto') {
    document.documentElement.classList.add(theme)
  }
}

// Must sit inside AuthProvider: the saved theme comes from the signed-in user.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [theme, setThemeState] = useState<Theme>('auto')

  useEffect(() => {
    const next = user?.theme ?? 'auto'
    setThemeState(next)
    applyTheme(next)
  }, [user])

  const setTheme = (next: Theme) => {
    // Applied before the save resolves; a failed save keeps the local choice.
    setThemeState(next)
    applyTheme(next)
    void settingsApi.updateTheme(next).catch(() => {})
  }

  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be called within a ThemeProvider')
  return ctx
}

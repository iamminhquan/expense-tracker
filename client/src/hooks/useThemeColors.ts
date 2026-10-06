import { useMemo, useState, useSyncExternalStore } from 'react'

export interface ThemeColors {
  /** Changes on every theme switch; key a chart on it to rebuild with the new colors. */
  key: string
  /** True once the theme has changed since the caller mounted, in either direction. */
  switched: boolean
  surface: string
  grid: string
  tick: string
  ink: string
  expense: string
  income: string
}

const VARS = ['--c-surface', '--c-border', '--c-ink-muted', '--c-ink', '--c-expense', '--c-chart-income'] as const

function subscribe(onChange: () => void) {
  const observer = new MutationObserver(onChange)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  mql.addEventListener('change', onChange)
  return () => {
    observer.disconnect()
    mql.removeEventListener('change', onChange)
  }
}

function snapshot(): string {
  const style = getComputedStyle(document.documentElement)
  return VARS.map((name) => style.getPropertyValue(name).trim()).join('|')
}

// Chart.js paints on a canvas, so it can't follow the CSS variables a theme switch changes.
export function useThemeColors(): ThemeColors {
  const key = useSyncExternalStore(subscribe, snapshot)
  const [mountKey] = useState(key)
  const [switched, setSwitched] = useState(false)
  // Comparing against the mount-time key alone missed a switch back to the starting theme.
  if (key !== mountKey && !switched) setSwitched(true)
  return useMemo(() => {
    const [surface, grid, tick, ink, expense, income] = key.split('|').map((channels) => `rgb(${channels})`)
    return { key, switched, surface, grid, tick, ink, expense, income }
  }, [key, switched])
}

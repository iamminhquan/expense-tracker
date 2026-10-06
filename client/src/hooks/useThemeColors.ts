import { useMemo, useSyncExternalStore } from 'react'

export interface ThemeColors {
  /** Changes on every theme switch; key a chart on it to rebuild with the new colors. */
  key: string
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
  return useMemo(() => {
    const [surface, grid, tick, ink, expense, income] = key.split('|').map((channels) => `rgb(${channels})`)
    return { key, surface, grid, tick, ink, expense, income }
  }, [key])
}

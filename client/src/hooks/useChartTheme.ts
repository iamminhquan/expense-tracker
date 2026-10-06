import { useMemo, useSyncExternalStore } from 'react'

/*
 * Chart.js paints to a canvas, so a CSS variable that changes under it changes
 * nothing until the chart is given new colors. This reads the active theme's
 * palette off :root and re-reads it when the theme class flips or, with the
 * "auto" theme, the OS preference does.
 */
const VARS = {
  expense: ['--c-expense', '180 35 24'],
  income: ['--c-income', '47 125 91'],
  text: ['--c-ink-muted', '87 83 78'],
  textFaint: ['--c-ink-faint', '138 135 129'],
  grid: ['--c-border-list', '241 239 236'],
} as const

export type ChartTheme = Record<keyof typeof VARS, string>

const KEYS = Object.keys(VARS) as (keyof typeof VARS)[]

function subscribe(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange)
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'style'] })
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  media.addEventListener('change', onChange)
  return () => {
    observer.disconnect()
    media.removeEventListener('change', onChange)
  }
}

// A string, so useSyncExternalStore can tell "same palette" from "new palette" by value.
function snapshot(): string {
  const styles = getComputedStyle(document.documentElement)
  return KEYS.map((k) => styles.getPropertyValue(VARS[k][0]).trim()).join('|')
}

export function useChartTheme(): ChartTheme {
  const raw = useSyncExternalStore(subscribe, snapshot, () => '')
  return useMemo(() => {
    const channels = raw.split('|')
    return Object.fromEntries(KEYS.map((k, i) => [k, `rgb(${channels[i] || VARS[k][1]})`])) as ChartTheme
  }, [raw])
}

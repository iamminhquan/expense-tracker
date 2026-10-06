import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useChartTheme } from './useChartTheme'

const root = document.documentElement

// jsdom has no matchMedia; this one lets a test flip the OS color scheme.
let osChanged: (() => void) | undefined

beforeEach(() => {
  osChanged = undefined
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({
      matches: false,
      addEventListener: (_: string, fn: () => void) => (osChanged = fn),
      removeEventListener: () => (osChanged = undefined),
    })),
  )
})

afterEach(() => {
  root.removeAttribute('style')
  root.className = ''
})

describe('useChartTheme', () => {
  it('falls back to the light palette when no theme variables are defined', () => {
    const { result } = renderHook(() => useChartTheme())
    expect(result.current.expense).toBe('rgb(180 35 24)')
    expect(result.current.income).toBe('rgb(47 125 91)')
  })

  it('reads the active palette off :root', () => {
    root.style.setProperty('--c-expense', '249 112 102')
    const { result } = renderHook(() => useChartTheme())
    expect(result.current.expense).toBe('rgb(249 112 102)')
  })

  // The dashboard's whole bug: flipping the theme left the canvas on the old colors.
  it('follows a theme change made after the chart mounted', async () => {
    root.style.setProperty('--c-expense', '180 35 24')
    const { result } = renderHook(() => useChartTheme())
    expect(result.current.expense).toBe('rgb(180 35 24)')

    act(() => {
      root.classList.add('dark')
      root.style.setProperty('--c-expense', '249 112 102')
    })

    await waitFor(() => expect(result.current.expense).toBe('rgb(249 112 102)'))
  })

  // The "auto" theme sets no class: the stylesheet's prefers-color-scheme query does the switching.
  it('follows the OS color scheme changing under the auto theme', async () => {
    root.style.setProperty('--c-income', '47 125 91')
    const { result } = renderHook(() => useChartTheme())

    root.style.setProperty('--c-income', '87 195 152')
    act(() => osChanged?.())

    await waitFor(() => expect(result.current.income).toBe('rgb(87 195 152)'))
  })

  it('stops listening when the chart unmounts', () => {
    const { unmount } = renderHook(() => useChartTheme())
    expect(osChanged).toBeDefined()
    unmount()
    expect(osChanged).toBeUndefined()
  })

  it('keeps the same object while the palette is unchanged, so charts are not rebuilt', () => {
    const { result, rerender } = renderHook(() => useChartTheme())
    const first = result.current
    rerender()
    expect(result.current).toBe(first)
  })
})

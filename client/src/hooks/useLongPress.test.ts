import { act, renderHook } from '@testing-library/react'
import type { PointerEvent } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useLongPress } from './useLongPress'

const pointer = (x: number, y: number) => ({ clientX: x, clientY: y }) as PointerEvent

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('useLongPress', () => {
  it('fires once the pointer has been held for the threshold', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => void vi.advanceTimersByTime(499))
    expect(onLongPress).not.toHaveBeenCalled()
    act(() => void vi.advanceTimersByTime(1))
    expect(onLongPress).toHaveBeenCalledTimes(1)
  })

  it('does not fire if the pointer is released first', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => void vi.advanceTimersByTime(300))
    act(() => result.current.onPointerUp())
    act(() => void vi.advanceTimersByTime(1000))
    expect(onLongPress).not.toHaveBeenCalled()
  })

  it('does not fire if the pointer is cancelled', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => result.current.onPointerCancel())
    act(() => void vi.advanceTimersByTime(1000))
    expect(onLongPress).not.toHaveBeenCalled()
  })

  it('treats a drag past the tolerance as a scroll, not a press', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => result.current.onPointerMove(pointer(0, 11)))
    act(() => void vi.advanceTimersByTime(1000))
    expect(onLongPress).not.toHaveBeenCalled()
  })

  it('tolerates the small jitter of a finger resting on the screen', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => result.current.onPointerMove(pointer(6, -9)))
    act(() => void vi.advanceTimersByTime(500))
    expect(onLongPress).toHaveBeenCalledTimes(1)
  })

  it('honours a custom threshold', () => {
    const onLongPress = vi.fn()
    const { result } = renderHook(() => useLongPress(onLongPress, { threshold: 200 }))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => void vi.advanceTimersByTime(200))
    expect(onLongPress).toHaveBeenCalledTimes(1)
  })
})

import { act, renderHook } from '@testing-library/react'
import type { MouseEvent, PointerEvent } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useLongPress } from './useLongPress'

const pointer = (x: number, y: number) => ({ clientX: x, clientY: y }) as PointerEvent
const click = () => ({ preventDefault: vi.fn(), stopPropagation: vi.fn() }) as unknown as MouseEvent & { stopPropagation: ReturnType<typeof vi.fn> }

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

  /* On a real touch the finger lifting fires a click, which lands on whatever the press just opened:
     a modal sheet's own <dialog>, read as a backdrop tap that closed it again. */
  it('swallows the click that follows a long-press', () => {
    const { result } = renderHook(() => useLongPress(vi.fn()))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => void vi.advanceTimersByTime(500))
    act(() => result.current.onPointerUp())
    const after = click()
    act(() => result.current.onClickCapture(after))
    expect(after.stopPropagation).toHaveBeenCalled()

    const next = click()
    act(() => result.current.onClickCapture(next))
    expect(next.stopPropagation).not.toHaveBeenCalled()
  })

  it('lets an ordinary tap through', () => {
    const { result } = renderHook(() => useLongPress(vi.fn()))

    act(() => result.current.onPointerDown(pointer(0, 0)))
    act(() => void vi.advanceTimersByTime(200))
    act(() => result.current.onPointerUp())
    const tap = click()
    act(() => result.current.onClickCapture(tap))
    expect(tap.stopPropagation).not.toHaveBeenCalled()
  })
})

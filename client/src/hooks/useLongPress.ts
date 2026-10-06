import { useRef } from 'react'
import type { PointerEvent } from 'react'

export function useLongPress(onLongPress: () => void, options: { threshold?: number; moveTolerance?: number } = {}) {
  const { threshold = 500, moveTolerance = 10 } = options
  const timerRef = useRef<number | null>(null)
  const startRef = useRef({ x: 0, y: 0 })

  function clear() {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  function onPointerDown(e: PointerEvent) {
    startRef.current = { x: e.clientX, y: e.clientY }
    timerRef.current = window.setTimeout(() => {
      onLongPress()
      timerRef.current = null
    }, threshold)
  }

  function onPointerMove(e: PointerEvent) {
    if (timerRef.current === null) return
    if (Math.abs(e.clientX - startRef.current.x) > moveTolerance || Math.abs(e.clientY - startRef.current.y) > moveTolerance) {
      clear()
    }
  }

  return {
    onPointerDown,
    onPointerMove,
    onPointerUp: clear,
    onPointerCancel: clear,
  }
}

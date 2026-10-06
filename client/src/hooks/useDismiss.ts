import { useEffect, type RefObject } from 'react'

// Closes a popover on Escape or on a pointer-down outside `ref`, then hands focus back to the trigger.
export function useDismiss(open: boolean, ref: RefObject<HTMLElement | null>, onClose: () => void, triggerRef?: RefObject<HTMLElement | null>) {
  useEffect(() => {
    if (!open) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      onClose()
      triggerRef?.current?.focus()
    }
    function onPointerDown(e: PointerEvent) {
      const target = e.target as Node
      if (ref.current?.contains(target) || triggerRef?.current?.contains(target)) return
      onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('pointerdown', onPointerDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('pointerdown', onPointerDown)
    }
  }, [open, ref, onClose, triggerRef])
}

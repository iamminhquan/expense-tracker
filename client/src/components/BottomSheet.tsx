import { useEffect, useRef, useState, type PointerEvent, type ReactNode } from 'react'

interface BottomSheetProps {
  open: boolean
  onClose: () => void
  children: ReactNode
}

// Mirrors server/internal/web/static/app.js's bottom-sheet drag-to-dismiss
// IIFE: drag the grab handle down to dismiss, past a quarter of the
// sheet's height or a short flick, otherwise it snaps back. A real
// <dialog> (showModal()) rather than a styled <div>, for the same reason
// the HTML side uses one -- native focus trapping, Escape-to-close, and a
// backdrop with no extra markup.
export function BottomSheet({ open, onClose, children }: BottomSheetProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const sheetRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ startY: number; startedAt: number } | null>(null)
  const [dragY, setDragY] = useState(0)
  const [dragging, setDragging] = useState(false)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    if (open && !dialog.open) {
      dialog.showModal()
      setDragY(0)
    } else if (!open && dialog.open) {
      dialog.close()
    }
  }, [open])

  function onHandlePointerDown(e: PointerEvent<HTMLDivElement>) {
    dragRef.current = { startY: e.clientY, startedAt: Date.now() }
    setDragging(true)
    e.currentTarget.setPointerCapture(e.pointerId)
  }

  function onHandlePointerMove(e: PointerEvent<HTMLDivElement>) {
    if (!dragRef.current) return
    // Downward only: dragging up must not lift the sheet off the bottom.
    setDragY(Math.max(0, e.clientY - dragRef.current.startY))
  }

  function releaseDrag() {
    if (!dragRef.current) return
    const height = sheetRef.current?.getBoundingClientRect().height ?? 0
    const { startedAt } = dragRef.current
    const moved = dragY
    dragRef.current = null
    setDragging(false)
    // Past a quarter of the sheet, or a short flick meant as one.
    if (moved > height * 0.25 || (moved > 40 && Date.now() - startedAt < 250)) {
      onClose()
    } else {
      setDragY(0)
    }
  }

  return (
    <dialog
      ref={dialogRef}
      onClose={onClose}
      onCancel={onClose}
      onClick={(e) => {
        // A click on the ::backdrop lands on the <dialog> element itself
        // (the only thing filling the viewport outside the sheet's own
        // box), never on a descendant -- which is what tells a backdrop
        // click apart from a click inside the sheet.
        if (e.target === dialogRef.current) onClose()
      }}
      className="fixed inset-x-0 top-auto bottom-0 m-0 w-full max-w-none rounded-t-[20px] border-0 bg-surface p-0 backdrop:bg-black/40"
    >
      <div
        ref={sheetRef}
        style={{
          transform: `translateY(${dragY}px)`,
          transition: dragging ? 'none' : 'transform 180ms ease-out',
        }}
      >
        <div
          onPointerDown={onHandlePointerDown}
          onPointerMove={onHandlePointerMove}
          onPointerUp={releaseDrag}
          onPointerCancel={releaseDrag}
          className="mx-auto my-3 h-1.5 w-10 touch-none rounded-full bg-border-nav"
          aria-hidden="true"
        />
        <div className="px-4 pb-6">{children}</div>
      </div>
    </dialog>
  )
}

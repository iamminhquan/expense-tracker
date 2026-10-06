import { useEffect, useRef, useState, type PointerEvent, type ReactNode } from 'react'

interface BottomSheetProps {
  open: boolean
  onClose: () => void
  /** Names the dialog for screen readers, since the sheet has no visible title of its own. */
  label: string
  children: ReactNode
}

const EXIT_MS = 200

// A native <dialog> gives focus trapping, Escape-to-close and the backdrop for free.
export function BottomSheet({ open, onClose, label, children }: BottomSheetProps) {
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
      return
    }
    if (!open && dialog.open) {
      const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      const timer = window.setTimeout(() => {
        dialog.close()
        setDragY(0)
      }, reduce ? 0 : EXIT_MS)
      return () => window.clearTimeout(timer)
    }
  }, [open])

  function onHandlePointerDown(e: PointerEvent<HTMLDivElement>) {
    dragRef.current = { startY: e.clientY, startedAt: Date.now() }
    setDragging(true)
    e.currentTarget.setPointerCapture(e.pointerId)
  }

  function onHandlePointerMove(e: PointerEvent<HTMLDivElement>) {
    if (!dragRef.current) return
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

  // While closing, `open` is already false but the dialog stays up for the exit slide.
  const leaving = !open
  const transform = leaving ? 'translateY(100%)' : `translateY(${dragY}px)`
  const transition = dragging
    ? 'none'
    : leaving
      ? `transform ${EXIT_MS}ms var(--ease-in)`
      : 'transform 240ms var(--ease-sheet)'

  return (
    <dialog
      ref={dialogRef}
      aria-label={label}
      onClose={onClose}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      onClick={(e) => {
        // A backdrop click targets the <dialog> itself, never a descendant.
        if (e.target === dialogRef.current) onClose()
      }}
      className="fixed inset-x-0 top-auto bottom-0 m-0 w-full max-w-none overflow-visible border-0 bg-transparent p-0"
    >
      <div
        ref={sheetRef}
        style={{ transform, transition }}
        className="animate-sheet-in rounded-t-[28px] border border-b-0 border-border bg-surface px-4 pt-2.5 pb-7 shadow-popover"
      >
        <div
          onPointerDown={onHandlePointerDown}
          onPointerMove={onHandlePointerMove}
          onPointerUp={releaseDrag}
          onPointerCancel={releaseDrag}
          className="-mx-4 -mt-2.5 flex h-8 touch-none items-center justify-center"
          aria-hidden="true"
        >
          <span className="h-[5px] w-11 rounded-full bg-border-strong" />
        </div>
        {children}
      </div>
    </dialog>
  )
}

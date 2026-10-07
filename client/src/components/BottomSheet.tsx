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
        className="flex max-h-[92dvh] animate-sheet-in flex-col rounded-t-[28px] bg-surface shadow-popover"
      >
        <div
          onPointerDown={onHandlePointerDown}
          onPointerMove={onHandlePointerMove}
          onPointerUp={releaseDrag}
          onPointerCancel={releaseDrag}
          className="flex h-9 shrink-0 cursor-grab touch-none items-center justify-center active:cursor-grabbing"
          aria-hidden="true"
        >
          <span className={`h-[5px] rounded-full bg-border-strong transition-[width] ${dragging ? 'w-14' : 'w-10'}`} />
        </div>
        <div className="overflow-y-auto overscroll-contain px-5 pb-[max(20px,env(safe-area-inset-bottom))]">{children}</div>
      </div>
    </dialog>
  )
}

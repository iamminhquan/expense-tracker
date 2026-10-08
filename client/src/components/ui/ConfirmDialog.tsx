import { useEffect, useId, useRef, type ReactNode } from 'react'
import { TriangleAlert } from 'lucide-react'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { buttonClass } from '../../lib/formStyles'

interface ConfirmDialogProps {
  open: boolean
  title: string
  children: ReactNode
  onCancel: () => void
  /** Leave out for a dialog that only explains why something can't be done. */
  onConfirm?: () => void
  confirmLabel?: string
  cancelLabel?: string
  pending?: boolean
  error?: string | null
}

export function ConfirmDialog({
  open,
  title,
  children,
  onCancel,
  onConfirm,
  confirmLabel = 'Delete',
  cancelLabel = onConfirm ? 'Cancel' : 'Got it',
  pending,
  error,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const isDesktop = useIsDesktop()
  const titleId = useId()
  const bodyId = useId()

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    else if (!open && dialog.open) dialog.close()
  }, [open])

  const wide = isDesktop ? '' : 'w-full'
  const cancelButton = (
    <button type="button" autoFocus onClick={onCancel} className={`${buttonClass(onConfirm ? 'secondary' : 'primary')} ${wide} ${isDesktop ? '' : 'h-[52px]'}`}>
      {cancelLabel}
    </button>
  )
  const confirmButton = onConfirm && (
    <button
      type="button"
      onClick={onConfirm}
      disabled={pending}
      aria-busy={pending}
      className={`${buttonClass('danger')} ${wide} ${isDesktop ? '' : 'h-[52px]'}`}
    >
      {confirmLabel}
    </button>
  )

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      aria-describedby={bodyId}
      onClose={onCancel}
      onCancel={(e) => {
        e.preventDefault()
        onCancel()
      }}
      onClick={(e) => {
        if (e.target === dialogRef.current) onCancel()
      }}
      className={
        isDesktop
          ? 'm-auto w-[460px] max-w-[calc(100vw-32px)] animate-dialog-in rounded-hero border border-border bg-surface p-8 shadow-popover'
          : 'fixed inset-x-0 top-auto bottom-0 m-0 w-full max-w-none animate-sheet-in rounded-t-sheet border border-b-0 border-border bg-surface px-5 pt-3 pb-[max(1.5rem,env(safe-area-inset-bottom))] shadow-popover'
      }
    >
      {!isDesktop && <span aria-hidden="true" className="mx-auto mb-5 block h-1.5 w-11 rounded-full bg-border" />}
      <span className="mb-4 flex size-12 items-center justify-center rounded-[16px] bg-danger-tint text-danger" aria-hidden="true">
        <TriangleAlert className="size-6" />
      </span>
      <h2 id={titleId} className="font-display text-[24px] leading-[30px] font-semibold tracking-[-0.02em] text-ink">
        {title}
      </h2>
      <div id={bodyId} className="mt-2 text-[15px] leading-[23px] text-ink-muted">
        {children}
      </div>
      {error && (
        <p role="alert" className="mt-3 text-[14px] font-semibold text-danger">
          {error}
        </p>
      )}
      <div className={`mt-7 flex gap-3 ${isDesktop ? 'justify-end' : 'flex-col'}`}>
        {isDesktop ? (
          <>
            {cancelButton}
            {confirmButton}
          </>
        ) : (
          <>
            {confirmButton}
            {cancelButton}
          </>
        )}
      </div>
    </dialog>
  )
}

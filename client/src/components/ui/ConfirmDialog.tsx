import { useEffect, useId, useRef, type ReactNode } from 'react'
import { Trash2, TriangleAlert } from 'lucide-react'
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

  const size = isDesktop ? 'md' : 'lg'
  const cancelButton = (
    <button type="button" autoFocus onClick={onCancel} className={`${buttonClass(onConfirm ? 'secondary' : 'primary', size)} ${isDesktop ? '' : 'w-full'}`}>
      {cancelLabel}
    </button>
  )
  const confirmButton = onConfirm && (
    <button type="button" onClick={onConfirm} disabled={pending} aria-busy={pending} className={`${buttonClass('danger', size)} ${isDesktop ? '' : 'w-full'}`}>
      <Trash2 aria-hidden="true" />
      {pending ? 'Deleting…' : confirmLabel}
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
          ? 'm-auto w-[460px] max-w-[calc(100vw-32px)] animate-dialog-in rounded-tile border border-border bg-surface p-7 shadow-popover'
          : 'fixed inset-x-0 top-auto bottom-0 m-0 w-full max-w-none animate-sheet-in rounded-t-[28px] border-0 bg-surface px-5 pt-3 pb-[max(20px,env(safe-area-inset-bottom))] shadow-popover'
      }
    >
      {!isDesktop && <span aria-hidden="true" className="mx-auto mb-5 block h-[5px] w-10 rounded-full bg-border" />}
      <div className={isDesktop ? 'flex gap-4' : ''}>
        <span
          aria-hidden="true"
          className={`flex size-12 shrink-0 items-center justify-center rounded-[16px] ${onConfirm ? 'bg-danger-tint text-danger' : 'bg-warning-tint text-warning'} ${isDesktop ? '' : 'mb-4'}`}
        >
          {onConfirm ? <Trash2 className="size-[22px]" /> : <TriangleAlert className="size-[22px]" />}
        </span>
        <div className="min-w-0">
          <h2 id={titleId} className="heading text-[21px] leading-7 text-ink">
            {title}
          </h2>
          <div id={bodyId} className="mt-1.5 text-[15px] leading-[22px] text-ink-muted">
            {children}
          </div>
          {error && (
            <p role="alert" className="mt-3 text-[14px] font-semibold text-danger">
              {error}
            </p>
          )}
        </div>
      </div>
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

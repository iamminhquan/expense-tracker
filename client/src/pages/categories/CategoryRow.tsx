import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Check, Lock, Pencil, Trash2 } from 'lucide-react'
import { useDeleteCategory, useUpdateCategory } from '../../hooks/useCategories'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { ApiError } from '../../lib/api/client'
import { buttonClass, iconButtonClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Badge } from '../../components/ui/Badge'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import type { Category } from '../../lib/api/types'

function countLabel(n: number): string {
  return `${n} transaction${n === 1 ? '' : 's'}`
}

export function CategoryRow({ category }: { category: Category }) {
  const updateCategory = useUpdateCategory()
  const deleteCategory = useDeleteCategory()
  const toast = useToast()
  const isDesktop = useIsDesktop()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(category.name)
  const [error, setError] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const renameButtonRef = useRef<HTMLButtonElement>(null)
  const wasEditing = useRef(false)

  useEffect(() => {
    if (wasEditing.current && !editing) renameButtonRef.current?.focus()
    wasEditing.current = editing
  }, [editing])

  // The server refuses this: there's no income-side "Other" to move the transactions to.
  const blocked = category.type === 'income' && category.transactionCount > 0

  function startRename() {
    setName(category.name)
    setError(null)
    setEditing(true)
  }

  function askDelete() {
    setDeleteError(null)
    setConfirmOpen(true)
  }

  async function onSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateCategory.mutateAsync({ id: category.id, input: { name: name.trim() } })
      setEditing(false)
      toast.success('Category renamed')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not rename the category.')
    }
  }

  async function onDelete() {
    try {
      await deleteCategory.mutateAsync(category.id)
      setConfirmOpen(false)
      toast.success(`Deleted "${category.name}"`)
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : 'Could not delete the category.')
    }
  }

  const dot = <span aria-hidden="true" className="size-3.5 shrink-0 rounded-full" style={{ backgroundColor: category.color }} />

  if (editing) {
    return (
      <li className="border-t border-border bg-surface-2 px-4 py-3 first:border-t-0 md:px-5">
        <form
          onSubmit={onSave}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setEditing(false)
          }}
          className="flex flex-wrap items-center gap-3"
        >
          {dot}
          <input
            autoFocus
            required
            maxLength={40}
            aria-label={`New name for ${category.name}`}
            aria-invalid={error ? true : undefined}
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={`${inputClass} min-w-0 flex-1`}
          />
          <div className="flex gap-1">
            <button type="submit" disabled={updateCategory.isPending} aria-busy={updateCategory.isPending} className={buttonClass('primary')}>
              <Check aria-hidden="true" />
              Save
            </button>
            <button type="button" onClick={() => setEditing(false)} className={buttonClass('ghost')}>
              Cancel
            </button>
          </div>
        </form>
        {error && (
          <div role="alert">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
      </li>
    )
  }

  return (
    <li className="flex min-h-14 items-center gap-3 border-t border-border px-4 py-1.5 first:border-t-0 md:px-5">
      {dot}
      <div className="min-w-0 flex-1">
        <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{category.name}</p>
        <p className="tabular text-[13px] leading-[18px] text-ink-muted">{countLabel(category.transactionCount)}</p>
      </div>
      {category.isDefault ? (
        <Badge icon={<Lock aria-hidden="true" />}>Default</Badge>
      ) : isDesktop ? (
        <div className="flex gap-1">
          <button
            ref={renameButtonRef}
            type="button"
            onClick={startRename}
            aria-label={`Rename ${category.name}`}
            className={buttonClass('ghost', 'sm')}
          >
            <Pencil aria-hidden="true" />
            Rename
          </button>
          <button type="button" onClick={askDelete} aria-label={`Delete ${category.name}`} className={buttonClass('danger-ghost', 'sm')}>
            <Trash2 aria-hidden="true" />
            Delete
          </button>
        </div>
      ) : (
        <div className="flex">
          <button
            ref={renameButtonRef}
            type="button"
            onClick={startRename}
            aria-label={`Rename ${category.name}`}
            className={iconButtonClass}
          >
            <Pencil aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={askDelete}
            aria-label={`Delete ${category.name}`}
            className={`${iconButtonClass} hover:bg-danger-tint hover:text-danger`}
          >
            <Trash2 aria-hidden="true" />
          </button>
        </div>
      )}

      {blocked ? (
        <ConfirmDialog open={confirmOpen} title={`"${category.name}" can't be deleted yet`} onCancel={() => setConfirmOpen(false)}>
          {countLabel(category.transactionCount)} use it, and income has no "Other" category to move them to. Move or delete those transactions first.
        </ConfirmDialog>
      ) : (
        <ConfirmDialog
          open={confirmOpen}
          title={`Delete "${category.name}"?`}
          onCancel={() => setConfirmOpen(false)}
          onConfirm={() => void onDelete()}
          pending={deleteCategory.isPending}
          error={deleteError}
        >
          {category.transactionCount > 0
            ? `Its ${countLabel(category.transactionCount)} will move to "Other". This can't be undone.`
            : "It isn't used by any transactions. This can't be undone."}
        </ConfirmDialog>
      )}
    </li>
  )
}

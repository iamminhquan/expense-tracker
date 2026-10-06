import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Check, Pencil, Trash2, TriangleAlert } from 'lucide-react'
import { useDeleteTransaction, useUpdateTransaction } from '../../hooks/useTransactions'
import { useLongPress } from '../../hooks/useLongPress'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { BottomSheet } from '../../components/BottomSheet'
import { AmountInput } from '../../components/ui/AmountInput'
import { Badge } from '../../components/ui/Badge'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { SelectControl } from '../../components/ui/SelectControl'
import { ApiError } from '../../lib/api/client'
import { buttonClass, iconButtonClass, inputClass } from '../../lib/formStyles'
import { formatDateLong, formatDateShort, formatVNDSigned } from '../../lib/format'
import { useToast } from '../../lib/toast/ToastContext'
import type { Category, Transaction } from '../../lib/api/types'
import { DESKTOP_EDIT_GRID, DESKTOP_ROW_GRID } from './rowLayout'

interface TransactionRowProps {
  transaction: Transaction
  showYear: boolean
  categories: Category[]
}

export function TransactionRow({ transaction, showYear, categories }: TransactionRowProps) {
  const updateTransaction = useUpdateTransaction()
  const deleteTransaction = useDeleteTransaction()
  const toast = useToast()
  const isDesktop = useIsDesktop()
  const [editing, setEditing] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [amount, setAmount] = useState(String(transaction.amount))
  const [categoryId, setCategoryId] = useState(transaction.categoryId)
  const [occurredOn, setOccurredOn] = useState(transaction.occurredOn)
  const [description, setDescription] = useState(transaction.description)
  const [error, setError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const editButtonRef = useRef<HTMLButtonElement>(null)
  const wasEditing = useRef(false)

  // Long-press adds to the Edit/Delete buttons below; it never replaces them.
  const longPress = useLongPress(() => setSheetOpen(true))

  // The Edit button only exists again after the re-render, so focus it from an effect.
  useEffect(() => {
    if (wasEditing.current && !editing) editButtonRef.current?.focus()
    wasEditing.current = editing
  }, [editing])

  const rowName = transaction.description || transaction.categoryName
  const sameTypeCategories = categories.filter((c) => c.type === transaction.type)
  const date = showYear ? formatDateLong(transaction.occurredOn) : formatDateShort(transaction.occurredOn)
  const signed = formatVNDSigned(transaction.type === 'expense' ? -transaction.amount : transaction.amount)
  const amountClass = transaction.type === 'income' ? 'text-income' : 'text-ink'

  function startEditing() {
    setError(null)
    setAmount(String(transaction.amount))
    setCategoryId(transaction.categoryId)
    setOccurredOn(transaction.occurredOn)
    setDescription(transaction.description)
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
      await updateTransaction.mutateAsync({ id: transaction.id, input: { categoryId, amount: Number(amount), occurredOn, description } })
      setEditing(false)
      toast.success('Changes saved')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update the transaction.')
    }
  }

  async function onDelete() {
    try {
      await deleteTransaction.mutateAsync(transaction.id)
      setConfirmOpen(false)
      toast.success('Transaction deleted')
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : 'Could not delete the transaction.')
    }
  }

  const dot = <span aria-hidden="true" className="size-3 shrink-0 rounded-full" style={{ backgroundColor: transaction.categoryColor }} />
  const duplicateBadge = transaction.isDuplicate && (
    <Badge kind="warning" icon={<TriangleAlert aria-hidden="true" />}>
      Possible duplicate
    </Badge>
  )

  const confirm = (
    <ConfirmDialog
      open={confirmOpen}
      title="Delete this transaction?"
      onCancel={() => setConfirmOpen(false)}
      onConfirm={() => void onDelete()}
      pending={deleteTransaction.isPending}
      error={deleteError}
    >
      <span className="font-semibold text-ink">{rowName}</span> · <span className="tabular">{signed}</span> on {formatDateLong(transaction.occurredOn)}. This
      can't be undone.
    </ConfirmDialog>
  )

  if (editing) {
    const categorySelect = (props: object) => (
      <SelectControl {...props} value={categoryId} onChange={(e) => setCategoryId(Number(e.target.value))}>
        {sameTypeCategories.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </SelectControl>
    )
    const cancel = (
      <button type="button" onClick={() => setEditing(false)} className={buttonClass('ghost')}>
        Cancel
      </button>
    )
    const save = (
      <button type="submit" disabled={updateTransaction.isPending} aria-busy={updateTransaction.isPending} className={buttonClass('primary')}>
        <Check aria-hidden="true" />
        Save
      </button>
    )

    return (
      <li className="animate-fade-in border-t border-border bg-surface-2 first:border-t-0">
        <form
          onSubmit={onSave}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setEditing(false)
          }}
          aria-label={`Edit ${rowName}`}
          className={isDesktop ? `${DESKTOP_EDIT_GRID} px-5 py-3` : 'grid gap-3 p-4'}
        >
          {isDesktop ? (
            <>
              <input autoFocus aria-label="Note" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Note" className={inputClass} />
              {categorySelect({ 'aria-label': 'Category' })}
              <input type="date" required aria-label="Date" value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />
              <AmountInput required aria-label="Amount" value={amount} onChange={(e) => setAmount(e.target.value)} />
              <div className="flex gap-1">
                {save}
                {cancel}
              </div>
            </>
          ) : (
            <>
              <Field label="Note">
                {(control) => <input {...control} autoFocus value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />}
              </Field>
              <Field label="Category">{(control) => categorySelect(control)}</Field>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Date">
                  {(control) => <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />}
                </Field>
                <Field label="Amount">
                  {(control) => <AmountInput {...control} required value={amount} onChange={(e) => setAmount(e.target.value)} />}
                </Field>
              </div>
              <div className="grid grid-cols-2 gap-3">
                {cancel}
                {save}
              </div>
            </>
          )}
        </form>
        {error && (
          <div role="alert" className="px-5 pb-3">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
      </li>
    )
  }

  if (isDesktop) {
    return (
      <li className={`${DESKTOP_ROW_GRID} min-h-[57px] border-t border-border px-5 py-2.5 first:border-t-0 hover:bg-[color-mix(in_srgb,var(--color-surface),var(--color-ink)_4%)]`}>
        <div className="flex min-w-0 items-center gap-3">
          {dot}
          <span className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</span>
          {duplicateBadge}
        </div>
        <span className="truncate text-[14px] leading-5 text-ink-muted">{transaction.categoryName}</span>
        <span className="tabular text-[14px] leading-5 whitespace-nowrap text-ink-muted">{date}</span>
        <span className={`tabular text-right font-display text-[16px] leading-[22px] font-bold whitespace-nowrap ${amountClass}`}>{signed}</span>
        <div className="flex justify-end gap-1">
          <button ref={editButtonRef} type="button" onClick={startEditing} aria-label={`Edit ${rowName}`} className={buttonClass('ghost', 'sm')}>
            <Pencil aria-hidden="true" />
            Edit
          </button>
          <button type="button" onClick={askDelete} aria-label={`Delete ${rowName}`} className={buttonClass('danger-ghost', 'sm')}>
            <Trash2 aria-hidden="true" />
            Delete
          </button>
        </div>
        {confirm}
      </li>
    )
  }

  return (
    <li className="border-t border-border px-4 pt-3 pb-1 select-none first:border-t-0 active:bg-surface-2" {...longPress}>
      <div className="flex items-center gap-3">
        {dot}
        <span className="min-w-0 flex-1 truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</span>
        <span className={`tabular font-display text-[16px] leading-[22px] font-bold whitespace-nowrap ${amountClass}`}>{signed}</span>
      </div>
      {duplicateBadge && <div className="mt-1.5 pl-6">{duplicateBadge}</div>}
      <div className="flex items-center gap-1 pl-6">
        <span className="tabular min-w-0 flex-1 truncate text-[13px] leading-[18px] text-ink-muted">
          {transaction.categoryName} · {date}
        </span>
        <button ref={editButtonRef} type="button" onClick={startEditing} aria-label={`Edit ${rowName}`} className={iconButtonClass}>
          <Pencil aria-hidden="true" />
        </button>
        <button type="button" onClick={askDelete} aria-label={`Delete ${rowName}`} className={`${iconButtonClass} hover:bg-danger-tint hover:text-danger`}>
          <Trash2 aria-hidden="true" />
        </button>
      </div>

      <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)} label={`Actions for ${rowName}`}>
        <div className="flex items-center gap-3 px-1 pt-1 pb-5">
          {dot}
          <div className="min-w-0 flex-1">
            <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</p>
            <p className="tabular text-[13px] leading-[18px] text-ink-muted">
              {transaction.categoryName} · {date}
            </p>
          </div>
          <span className={`tabular font-display text-[17px] font-bold whitespace-nowrap ${amountClass}`}>{signed}</span>
        </div>
        <div className="grid gap-2.5">
          <button
            type="button"
            onClick={() => {
              setSheetOpen(false)
              startEditing()
            }}
            className={`${buttonClass('secondary')} h-[52px] rounded-[14px]`}
          >
            <Pencil aria-hidden="true" />
            Edit
          </button>
          <button
            type="button"
            onClick={() => {
              setSheetOpen(false)
              askDelete()
            }}
            className={`${buttonClass('danger-tint')} h-[52px] rounded-[14px]`}
          >
            <Trash2 aria-hidden="true" />
            Delete
          </button>
        </div>
      </BottomSheet>
      {confirm}
    </li>
  )
}

import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Check, Ellipsis, Pencil, Trash2, TriangleAlert } from 'lucide-react'
import { useDeleteTransaction, useUpdateTransaction } from '../../hooks/useTransactions'
import { useLongPress } from '../../hooks/useLongPress'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { BottomSheet } from '../../components/BottomSheet'
import { AmountInput } from '../../components/ui/AmountInput'
import { Badge } from '../../components/ui/Badge'
import { CategoryAvatar } from '../../components/ui/CategoryAvatar'
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
  /** The mobile list puts the date in a heading above the rows, so the row leaves it out. */
  dateInHeading?: boolean
}

/* The sign is the signal; the pill and colour on income are extra. */
function Amount({ type, text, className = '' }: { type: Transaction['type']; text: string; className?: string }) {
  return (
    <span
      className={`num whitespace-nowrap ${
        type === 'income' ? 'rounded-full bg-income-tint px-2.5 py-0.5 text-income' : 'text-ink'
      } text-[16px] leading-[22px] font-bold ${className}`}
    >
      {text}
    </span>
  )
}

export function TransactionRow({ transaction, showYear, categories, dateInHeading }: TransactionRowProps) {
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

  // The "…" button and the long-press open the same sheet; neither is the only way in.
  const longPress = useLongPress(() => setSheetOpen(true))

  // The button only exists again after the re-render, so focus it from an effect.
  useEffect(() => {
    if (wasEditing.current && !editing) editButtonRef.current?.focus()
    wasEditing.current = editing
  }, [editing])

  const rowName = transaction.description || transaction.categoryName
  const sameTypeCategories = categories.filter((c) => c.type === transaction.type)
  const date = showYear ? formatDateLong(transaction.occurredOn) : formatDateShort(transaction.occurredOn)
  const signed = formatVNDSigned(transaction.type === 'expense' ? -transaction.amount : transaction.amount)
  const meta = dateInHeading ? transaction.categoryName : `${transaction.categoryName} · ${date}`

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

  const avatar = <CategoryAvatar name={transaction.categoryName} color={transaction.categoryColor} />
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
      <button type="button" onClick={() => setEditing(false)} className={buttonClass(isDesktop ? 'ghost' : 'secondary')}>
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
      <li className="animate-fade-in border-t border-border bg-accent-tint/50 first:border-t-0">
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
              <Field label="Amount">
                {(control) => <AmountInput {...control} large autoFocus required value={amount} onChange={(e) => setAmount(e.target.value)} />}
              </Field>
              <Field label="Note">
                {(control) => <input {...control} value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />}
              </Field>
              <Field label="Category">{(control) => categorySelect(control)}</Field>
              <Field label="Date">
                {(control) => <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />}
              </Field>
              <div className="grid grid-cols-2 gap-3 pt-1">
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
      <li className={`${DESKTOP_ROW_GRID} min-h-[64px] border-t border-border px-5 py-2.5 first:border-t-0 hover:bg-surface-2/60`}>
        <div className="flex min-w-0 items-center gap-3">
          {avatar}
          <span className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</span>
          {duplicateBadge}
        </div>
        <span className="truncate text-[14px] leading-5 text-ink-muted">{transaction.categoryName}</span>
        <span className="tabular text-[14px] leading-5 whitespace-nowrap text-ink-muted">{date}</span>
        <span className="text-right">
          <Amount type={transaction.type} text={signed} />
        </span>
        <div className="flex justify-end gap-1">
          <button ref={editButtonRef} type="button" onClick={startEditing} aria-label={`Edit ${rowName}`} className={`${buttonClass('ghost', 'sm')} text-ink-muted hover:text-ink`}>
            <Pencil aria-hidden="true" />
            Edit
          </button>
          <button type="button" onClick={askDelete} aria-label={`Delete ${rowName}`} className={`${buttonClass('ghost', 'sm')} text-ink-muted hover:bg-danger-tint hover:text-danger`}>
            <Trash2 aria-hidden="true" />
            Delete
          </button>
        </div>
        {confirm}
      </li>
    )
  }

  return (
    <li className="border-t border-border py-2.5 pr-1.5 pl-4 select-none first:border-t-0 active:bg-surface-2" {...longPress}>
      <div className="flex items-center gap-3">
        {avatar}
        <div className="min-w-0 flex-1">
          <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</p>
          <p className="tabular truncate text-[13px] leading-[18px] text-ink-muted">{meta}</p>
        </div>
        <Amount type={transaction.type} text={signed} />
        <button ref={editButtonRef} type="button" onClick={() => setSheetOpen(true)} aria-label={`Open actions for ${rowName}`} className={iconButtonClass}>
          <Ellipsis aria-hidden="true" />
        </button>
      </div>
      {duplicateBadge && <div className="mt-1.5 pl-14">{duplicateBadge}</div>}

      <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)} label={`Actions for ${rowName}`}>
        <div className="flex items-center gap-3 px-1 pt-1 pb-5">
          {avatar}
          <div className="min-w-0 flex-1">
            <p className="truncate text-[16px] leading-6 font-semibold text-ink">{rowName}</p>
            <p className="tabular text-[13px] leading-[18px] text-ink-muted">
              {transaction.categoryName} · {date}
            </p>
          </div>
          <Amount type={transaction.type} text={signed} className="text-[18px]" />
        </div>
        <div className="grid gap-2.5">
          <button
            type="button"
            onClick={() => {
              setSheetOpen(false)
              startEditing()
            }}
            className={`${buttonClass('secondary')} h-14 rounded-card text-[16px]`}
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
            className={`${buttonClass('danger-tint')} h-14 rounded-card text-[16px]`}
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

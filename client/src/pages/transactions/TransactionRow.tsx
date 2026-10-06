import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useDeleteTransaction, useUpdateTransaction } from '../../hooks/useTransactions'
import { useLongPress } from '../../hooks/useLongPress'
import { BottomSheet } from '../../components/BottomSheet'
import { ApiError } from '../../lib/api/client'
import { formatDateLong, formatDateShort, formatVNDSigned } from '../../lib/format'
import type { Category, Transaction } from '../../lib/api/types'

const editFieldClass = 'rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]'

interface TransactionRowProps {
  transaction: Transaction
  showYear: boolean
  categories: Category[]
}

export function TransactionRow({ transaction, showYear, categories }: TransactionRowProps) {
  const updateTransaction = useUpdateTransaction()
  const deleteTransaction = useDeleteTransaction()
  const [editing, setEditing] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [amount, setAmount] = useState(String(transaction.amount))
  const [categoryId, setCategoryId] = useState(transaction.categoryId)
  const [occurredOn, setOccurredOn] = useState(transaction.occurredOn)
  const [description, setDescription] = useState(transaction.description)
  const [error, setError] = useState<string | null>(null)
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

  function startEditing() {
    setError(null)
    setEditing(true)
  }

  async function onSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateTransaction.mutateAsync({
        id: transaction.id,
        input: { categoryId, amount: Number(amount), occurredOn, description },
      })
      setEditing(false)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update the transaction.')
    }
  }

  async function onDelete() {
    if (!confirm('Delete this transaction?')) return
    setError(null)
    try {
      await deleteTransaction.mutateAsync(transaction.id)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete the transaction.')
    }
  }

  if (editing) {
    return (
      <li className="p-3">
        <form
          onSubmit={onSave}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setEditing(false)
          }}
          aria-label={`Edit ${rowName}`}
          className="flex flex-wrap items-end gap-2"
        >
          <select
            autoFocus
            aria-label="Category"
            value={categoryId}
            onChange={(e) => setCategoryId(Number(e.target.value))}
            className={editFieldClass}
          >
            {sameTypeCategories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <input
            type="number"
            inputMode="numeric"
            required
            min={1}
            aria-label="Amount"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className={`w-[100px] ${editFieldClass}`}
          />
          <input
            type="date"
            required
            aria-label="Date"
            value={occurredOn}
            onChange={(e) => setOccurredOn(e.target.value)}
            className={editFieldClass}
          />
          <input
            aria-label="Note"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Note"
            className={`min-w-[120px] flex-1 ${editFieldClass}`}
          />
          <button
            type="submit"
            disabled={updateTransaction.isPending}
            className="text-[12px] font-semibold text-accent disabled:opacity-50"
          >
            Save
          </button>
          <button type="button" onClick={() => setEditing(false)} className="text-[12px] text-ink-faint">
            Cancel
          </button>
        </form>
        {error && (
          <p role="alert" className="mt-1 text-[12px] text-expense">
            {error}
          </p>
        )}
      </li>
    )
  }

  return (
    <li className="p-3 text-[13px] select-none" {...longPress}>
      <div className="flex items-center gap-3">
        <span className="w-[56px] shrink-0 text-ink-faint">
          {showYear ? formatDateLong(transaction.occurredOn) : formatDateShort(transaction.occurredOn)}
        </span>
        <span
          aria-hidden="true"
          className="h-2.5 w-2.5 shrink-0 rounded-full"
          style={{ backgroundColor: transaction.categoryColor }}
        />
        <span className="w-[100px] shrink-0 truncate text-ink-muted">{transaction.categoryName}</span>
        <span className="flex-1 truncate text-ink">
          {transaction.description}
          {transaction.isDuplicate && (
            <span className="ml-2 rounded-full bg-track px-2 py-0.5 text-[11px] text-ink-faint">possible duplicate</span>
          )}
        </span>
        <span className={`font-mono ${transaction.type === 'expense' ? 'text-expense' : 'text-income'}`}>
          {formatVNDSigned(transaction.type === 'expense' ? -transaction.amount : transaction.amount)}
        </span>
        <button
          ref={editButtonRef}
          type="button"
          onClick={startEditing}
          aria-label={`Edit ${rowName}`}
          className="text-ink-faint hover:text-ink"
        >
          Edit
        </button>
        <button
          type="button"
          onClick={() => void onDelete()}
          disabled={deleteTransaction.isPending}
          aria-label={`Delete ${rowName}`}
          className="text-ink-faint hover:text-expense disabled:opacity-50"
        >
          Delete
        </button>
      </div>
      {error && (
        <p role="alert" className="mt-1 text-[12px] text-expense">
          {error}
        </p>
      )}

      <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)}>
        <p className="mb-3 mt-1 truncate text-[13px] text-ink-muted">{rowName}</p>
        <div className="flex flex-col gap-1">
          <button
            type="button"
            onClick={() => {
              setSheetOpen(false)
              startEditing()
            }}
            className="rounded-[10px] px-3 py-2.5 text-left text-[14px] text-ink hover:bg-track"
          >
            Edit
          </button>
          <button
            type="button"
            onClick={() => {
              setSheetOpen(false)
              void onDelete()
            }}
            className="rounded-[10px] px-3 py-2.5 text-left text-[14px] text-expense hover:bg-expense/10"
          >
            Delete
          </button>
        </div>
      </BottomSheet>
    </li>
  )
}

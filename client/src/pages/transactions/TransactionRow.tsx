import { useState, type FormEvent } from 'react'
import { useDeleteTransaction, useUpdateTransaction } from '../../hooks/useTransactions'
import { useLongPress } from '../../hooks/useLongPress'
import { BottomSheet } from '../../components/BottomSheet'
import { ApiError } from '../../lib/api/client'
import { formatDateLong, formatDateShort, formatVNDSigned } from '../../lib/format'
import type { Category, Transaction } from '../../lib/api/types'

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

  // Long-press adds to the Edit/Delete buttons below; it never replaces them.
  const longPress = useLongPress(() => setSheetOpen(true))

  const sameTypeCategories = categories.filter((c) => c.type === transaction.type)

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
    try {
      await deleteTransaction.mutateAsync(transaction.id)
    } catch (err) {
      alert(err instanceof ApiError ? err.message : 'Could not delete the transaction.')
    }
  }

  if (editing) {
    return (
      <li className="p-3">
        <form onSubmit={onSave} className="flex flex-wrap items-end gap-2">
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(Number(e.target.value))}
            className="rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          >
            {sameTypeCategories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="w-[100px] rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          />
          <input
            type="date"
            value={occurredOn}
            onChange={(e) => setOccurredOn(e.target.value)}
            className="rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          />
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Note"
            className="min-w-[120px] flex-1 rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          />
          <button type="submit" className="text-[12px] font-semibold text-accent">
            Save
          </button>
          <button type="button" onClick={() => setEditing(false)} className="text-[12px] text-ink-faint">
            Cancel
          </button>
        </form>
        {error && <p className="mt-1 text-[12px] text-expense">{error}</p>}
      </li>
    )
  }

  return (
    <li className="flex items-center gap-3 p-3 text-[13px] select-none" {...longPress}>
      <span className="w-[56px] shrink-0 text-ink-faint">
        {showYear ? formatDateLong(transaction.occurredOn) : formatDateShort(transaction.occurredOn)}
      </span>
      <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: transaction.categoryColor }} />
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
      <button onClick={() => setEditing(true)} className="text-ink-faint hover:text-ink" aria-label="Edit">
        Edit
      </button>
      <button onClick={() => void onDelete()} className="text-ink-faint hover:text-expense" aria-label="Delete">
        Delete
      </button>

      <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)}>
        <p className="mb-3 mt-1 truncate text-[13px] text-ink-muted">{transaction.description || transaction.categoryName}</p>
        <div className="flex flex-col gap-1">
          <button
            onClick={() => {
              setSheetOpen(false)
              setEditing(true)
            }}
            className="rounded-[10px] px-3 py-2.5 text-left text-[14px] text-ink hover:bg-track"
          >
            Edit
          </button>
          <button
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

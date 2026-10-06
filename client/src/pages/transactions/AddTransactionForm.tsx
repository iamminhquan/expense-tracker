import { useState, type FormEvent } from 'react'
import { useCreateTransaction } from '../../hooks/useTransactions'
import { ApiError } from '../../lib/api/client'
import type { Category } from '../../lib/api/types'

const todayISO = () => new Date().toISOString().slice(0, 10)

interface AddTransactionFormProps {
  categories: Category[]
}

export function AddTransactionForm({ categories }: AddTransactionFormProps) {
  const createTransaction = useCreateTransaction()
  const [type, setType] = useState<'expense' | 'income'>('expense')
  const [categoryId, setCategoryId] = useState<number | ''>('')
  const [amount, setAmount] = useState('')
  const [occurredOn, setOccurredOn] = useState(todayISO())
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  const options = categories.filter((c) => c.type === type)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!categoryId) {
      setError('Please choose a category.')
      return
    }
    try {
      await createTransaction.mutateAsync({
        categoryId: Number(categoryId),
        amount: Number(amount),
        occurredOn,
        description,
        type,
      })
      setAmount('')
      setDescription('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the transaction.')
    }
  }

  return (
    <form onSubmit={onSubmit} className="rounded-[16px] border border-border-card bg-surface p-4">
      <div className="flex flex-wrap items-end gap-2">
        <div className="flex gap-1 rounded-[9px] bg-track p-[3px]">
          {(['expense', 'income'] as const).map((t) => (
            <button
              key={t}
              type="button"
              onClick={() => {
                setType(t)
                setCategoryId('')
              }}
              className={`rounded-[6px] px-3 py-1.5 text-[13px] capitalize ${
                type === t ? 'bg-surface font-semibold text-ink shadow-sm' : 'text-ink-faint'
              }`}
            >
              {t}
            </button>
          ))}
        </div>
        <select
          required
          value={categoryId}
          onChange={(e) => setCategoryId(Number(e.target.value))}
          className="rounded-[10px] border border-border-input bg-surface px-2 py-2 text-[13px]"
        >
          <option value="" disabled>
            Category
          </option>
          {options.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <input
          type="number"
          required
          min={1}
          placeholder="Amount"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          className="w-[120px] rounded-[10px] border border-border-input bg-surface px-2 py-2 text-[13px]"
        />
        <input
          type="date"
          required
          value={occurredOn}
          onChange={(e) => setOccurredOn(e.target.value)}
          className="rounded-[10px] border border-border-input bg-surface px-2 py-2 text-[13px]"
        />
        <input
          placeholder="Note (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className="min-w-[140px] flex-1 rounded-[10px] border border-border-input bg-surface px-2 py-2 text-[13px]"
        />
        <button
          type="submit"
          disabled={createTransaction.isPending}
          className="rounded-[10px] bg-accent px-4 py-2 text-[13px] font-semibold text-on-solid hover:opacity-90 disabled:opacity-50"
        >
          Add
        </button>
      </div>
      {error && <p className="mt-2 text-[13px] text-expense">{error}</p>}
    </form>
  )
}

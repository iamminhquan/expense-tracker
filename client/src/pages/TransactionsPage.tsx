import { useState } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { useCategories } from '../hooks/useCategories'
import { useCreateTransaction, useDeleteTransaction, useTransactions, useUpdateTransaction } from '../hooks/useTransactions'
import { MonthPicker } from '../components/MonthPicker'
import { ApiError } from '../lib/api/client'
import { downloadTransactionsExport } from '../lib/api/import'
import { formatDateLong, formatDateShort, formatVNDSigned } from '../lib/format'
import type { Transaction, TransactionFilters } from '../lib/api/types'

const todayISO = () => new Date().toISOString().slice(0, 10)

// The URL's own query string is the source of truth for every filter --
// not local component state -- so a reload, a bookmark, or (the bug a
// browser smoke test caught) the redirect after a CSV import landing on
// /transactions?month=2026-02 all show the right month instead of
// silently resetting to whatever the component's initial state happened
// to be.
function filtersFromSearchParams(params: URLSearchParams): TransactionFilters {
  const filters: TransactionFilters = {}
  if (params.get('month')) filters.month = params.get('month')!
  if (params.get('page')) filters.page = Number(params.get('page'))
  if (params.get('q')) filters.q = params.get('q')!
  const type = params.get('type')
  if (type === 'expense' || type === 'income') filters.type = type
  if (params.get('category')) filters.category = Number(params.get('category'))
  if (params.get('min')) filters.min = Number(params.get('min'))
  if (params.get('max')) filters.max = Number(params.get('max'))
  const sort = params.get('sort')
  if (sort === 'amount_desc' || sort === 'amount_asc') filters.sort = sort
  return filters
}

function exportQueryString(filters: TransactionFilters): string {
  const params = new URLSearchParams()
  if (filters.month) params.set('month', filters.month)
  if (filters.q) params.set('q', filters.q)
  if (filters.type) params.set('type', filters.type)
  if (filters.category) params.set('category', String(filters.category))
  if (filters.min) params.set('min', String(filters.min))
  if (filters.max) params.set('max', String(filters.max))
  if (filters.sort) params.set('sort', filters.sort)
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export function TransactionsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = filtersFromSearchParams(searchParams)
  const { data, isLoading, error } = useTransactions(filters)
  const { data: categories } = useCategories()

  function setFilter<K extends keyof TransactionFilters>(key: K, value: TransactionFilters[K]) {
    const next = new URLSearchParams(searchParams)
    if (value === undefined || value === '') {
      next.delete(key)
    } else {
      next.set(key, String(value))
    }
    // Any filter change other than paging itself goes back to page 1 --
    // the page a changed filter's results actually start on.
    if (key !== 'page') next.delete('page')
    setSearchParams(next, { replace: true })
  }

  if (isLoading || !data) return <p className="text-ink-faint">Loading…</p>
  if (error) return <p className="text-expense">Could not load transactions.</p>

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-[20px] font-semibold">Transactions</h1>
        <div className="flex items-center gap-3">
          <button
            onClick={() => void downloadTransactionsExport(exportQueryString(filters))}
            className="text-[13px] text-ink-faint hover:text-ink"
          >
            Export CSV
          </button>
          <Link to="/transactions/import" className="text-[13px] text-ink-faint hover:text-ink">
            Import CSV
          </Link>
          <MonthPicker
            value={data.monthValue}
            label={data.monthLabel}
            currentMonthValue={data.currentMonthValue}
            availableMonths={data.availableMonths}
            onChange={(v) => setFilter('month', v)}
            allowAllMonths
          />
        </div>
      </div>

      <AddTransactionForm categories={categories?.expenseCategories.concat(categories?.incomeCategories ?? []) ?? []} />

      <FilterBar
        filters={filters}
        categories={categories?.expenseCategories.concat(categories?.incomeCategories ?? []) ?? []}
        onChange={setFilter}
      />

      <div className="rounded-[16px] border border-border-card bg-surface">
        {data.transactions.length === 0 ? (
          <p className="p-6 text-center text-[13px] text-ink-faint">
            {data.totalCount === 0 ? `No transactions in ${data.monthLabel.toLowerCase()}.` : 'Nothing matches your filters.'}
          </p>
        ) : (
          <ul className="divide-y divide-border-list">
            {data.transactions.map((t) => (
              <TransactionRow key={t.id} transaction={t} showYear={data.allMonths} categories={categories?.expenseCategories.concat(categories?.incomeCategories ?? []) ?? []} />
            ))}
          </ul>
        )}
      </div>

      {data.totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 text-[13px]">
          <button
            disabled={!data.hasPrev}
            onClick={() => setFilter('page', data.page - 1)}
            className="rounded-[8px] px-3 py-1.5 text-ink-muted hover:bg-track disabled:opacity-40"
          >
            Previous
          </button>
          <span className="text-ink-faint">
            Page {data.page} of {data.totalPages}
          </span>
          <button
            disabled={!data.hasNext}
            onClick={() => setFilter('page', data.page + 1)}
            className="rounded-[8px] px-3 py-1.5 text-ink-muted hover:bg-track disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

function FilterBar({
  filters,
  categories,
  onChange,
}: {
  filters: TransactionFilters
  categories: { id: number; name: string }[]
  onChange: <K extends keyof TransactionFilters>(key: K, value: TransactionFilters[K]) => void
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <input
        placeholder="Search…"
        value={filters.q ?? ''}
        onChange={(e) => onChange('q', e.target.value || undefined)}
        className="min-w-[140px] flex-1 rounded-[10px] border border-border-input bg-surface px-3 py-1.5 text-[13px]"
      />
      <select
        value={filters.type ?? ''}
        onChange={(e) => onChange('type', (e.target.value || undefined) as TransactionFilters['type'])}
        className="rounded-[10px] border border-border-input bg-surface px-2 py-1.5 text-[13px]"
      >
        <option value="">All types</option>
        <option value="expense">Expense</option>
        <option value="income">Income</option>
      </select>
      <select
        value={filters.category ?? ''}
        onChange={(e) => onChange('category', e.target.value ? Number(e.target.value) : undefined)}
        className="rounded-[10px] border border-border-input bg-surface px-2 py-1.5 text-[13px]"
      >
        <option value="">All categories</option>
        {categories.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
      <select
        value={filters.sort ?? ''}
        onChange={(e) => onChange('sort', (e.target.value || undefined) as TransactionFilters['sort'])}
        className="rounded-[10px] border border-border-input bg-surface px-2 py-1.5 text-[13px]"
      >
        <option value="">Newest first</option>
        <option value="amount_desc">Amount: high to low</option>
        <option value="amount_asc">Amount: low to high</option>
      </select>
    </div>
  )
}

function AddTransactionForm({ categories }: { categories: { id: number; name: string; type: string }[] }) {
  const createTransaction = useCreateTransaction()
  const [type, setType] = useState<'expense' | 'income'>('expense')
  const [categoryId, setCategoryId] = useState<number | ''>('')
  const [amount, setAmount] = useState('')
  const [occurredOn, setOccurredOn] = useState(todayISO())
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  const options = categories.filter((c) => c.type === type)

  async function onSubmit(e: React.FormEvent) {
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

function TransactionRow({
  transaction,
  showYear,
  categories,
}: {
  transaction: Transaction
  showYear: boolean
  categories: { id: number; name: string; type: string }[]
}) {
  const updateTransaction = useUpdateTransaction()
  const deleteTransaction = useDeleteTransaction()
  const [editing, setEditing] = useState(false)
  const [amount, setAmount] = useState(String(transaction.amount))
  const [categoryId, setCategoryId] = useState(transaction.categoryId)
  const [occurredOn, setOccurredOn] = useState(transaction.occurredOn)
  const [description, setDescription] = useState(transaction.description)
  const [error, setError] = useState<string | null>(null)

  const sameTypeCategories = categories.filter((c) => c.type === transaction.type)

  async function onSave(e: React.FormEvent) {
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
    <li className="flex items-center gap-3 p-3 text-[13px]">
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
    </li>
  )
}

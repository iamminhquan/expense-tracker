import { useSearchParams, Link } from 'react-router-dom'
import { useCategories } from '../../hooks/useCategories'
import { useTransactions } from '../../hooks/useTransactions'
import { MonthPicker } from '../../components/MonthPicker'
import { downloadTransactionsExport } from '../../lib/api/import'
import type { TransactionFilters } from '../../lib/api/types'
import { AddTransactionForm } from './AddTransactionForm'
import { FilterBar } from './FilterBar'
import { TransactionRow } from './TransactionRow'

// Filters live in the URL, not useState, so reloads, bookmarks and links keep them.
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
  const allCategories = categories ? categories.expenseCategories.concat(categories.incomeCategories) : []

  function setFilter<K extends keyof TransactionFilters>(key: K, value: TransactionFilters[K]) {
    const next = new URLSearchParams(searchParams)
    if (value === undefined || value === '') {
      next.delete(key)
    } else {
      next.set(key, String(value))
    }
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

      <AddTransactionForm categories={allCategories} />

      <FilterBar filters={filters} categories={allCategories} onChange={setFilter} />

      <div className="rounded-[16px] border border-border-card bg-surface">
        {data.transactions.length === 0 ? (
          <p className="p-6 text-center text-[13px] text-ink-faint">
            {data.totalCount === 0 ? `No transactions in ${data.monthLabel.toLowerCase()}.` : 'Nothing matches your filters.'}
          </p>
        ) : (
          <ul className="divide-y divide-border-list">
            {data.transactions.map((t) => (
              <TransactionRow key={t.id} transaction={t} showYear={data.allMonths} categories={allCategories} />
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

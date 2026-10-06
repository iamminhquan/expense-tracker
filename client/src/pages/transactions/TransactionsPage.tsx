import { useSearchParams, Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Download, Inbox, SearchX, Upload } from 'lucide-react'
import { useCategories } from '../../hooks/useCategories'
import { useTransactions } from '../../hooks/useTransactions'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { MonthPicker } from '../../components/MonthPicker'
import { EmptyState } from '../../components/ui/EmptyState'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { downloadTransactionsExport } from '../../lib/api/import'
import { buttonClass, pageTitleClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import type { TransactionFilters } from '../../lib/api/types'
import { AddTransactionForm } from './AddTransactionForm'
import { FilterBar } from './FilterBar'
import { DESKTOP_ROW_GRID } from './rowLayout'
import { TransactionRow } from './TransactionRow'

const FILTER_KEYS = ['q', 'type', 'category', 'min', 'max', 'sort'] as const

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
  const { data, error, refetch } = useTransactions(filters)
  const { data: categories } = useCategories()
  const isDesktop = useIsDesktop()
  const toast = useToast()
  const allCategories = categories ? categories.expenseCategories.concat(categories.incomeCategories) : []
  const hasFilters = FILTER_KEYS.some((key) => searchParams.has(key))

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

  function clearFilters() {
    const next = new URLSearchParams(searchParams)
    for (const key of FILTER_KEYS) next.delete(key)
    next.delete('page')
    setSearchParams(next, { replace: true })
  }

  async function onExport() {
    try {
      await downloadTransactionsExport(exportQueryString(filters))
    } catch {
      toast.error('Could not export transactions.')
    }
  }

  if (!data) {
    if (error) return <InlineError message="Could not load transactions." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading transactions…" />
  }

  return (
    <div className="space-y-4 md:space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className={pageTitleClass}>Transactions</h1>
        <div className="flex flex-wrap items-center gap-2">
          <Link to="/transactions/import" className={buttonClass('secondary')}>
            <Upload aria-hidden="true" />
            <span>
              Import<span className="max-sm:sr-only"> CSV</span>
            </span>
          </Link>
          <button type="button" onClick={() => void onExport()} className={buttonClass('secondary')}>
            <Download aria-hidden="true" />
            <span>
              Export<span className="max-sm:sr-only"> CSV</span>
            </span>
          </button>
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

      <FilterBar filters={filters} categories={allCategories} onChange={setFilter} onClear={clearFilters} />

      <section aria-label="Transaction list" className="overflow-hidden rounded-[24px] border border-border bg-surface md:rounded-[28px]">
        {data.transactions.length === 0 ? (
          hasFilters ? (
            <EmptyState
              icon={<SearchX />}
              title="Nothing matches your filters"
              actions={
                <button type="button" onClick={clearFilters} className={buttonClass('secondary')}>
                  Clear filters
                </button>
              }
            >
              Try a different search or remove a filter.
            </EmptyState>
          ) : (
            <EmptyState icon={<Inbox />} title={data.allMonths ? 'No transactions yet' : `No transactions in ${data.monthLabel}`}>
              Add one with the form above, or import a CSV from your bank.
            </EmptyState>
          )
        ) : (
          <>
            {isDesktop && (
              <div
                aria-hidden="true"
                className={`${DESKTOP_ROW_GRID} bg-surface-2 px-5 py-2.5 text-[12px] leading-4 font-semibold text-ink-muted`}
              >
                <span>Note</span>
                <span>Category</span>
                <span>Date</span>
                <span className="text-right">Amount</span>
                <span />
              </div>
            )}
            <ul>
              {data.transactions.map((t) => (
                <TransactionRow key={t.id} transaction={t} showYear={data.allMonths} categories={allCategories} />
              ))}
            </ul>
          </>
        )}
      </section>

      {data.totalPages > 1 && (
        <nav aria-label="Pages" className="flex items-center justify-between gap-3">
          <button type="button" disabled={!data.hasPrev} onClick={() => setFilter('page', data.page - 1)} className={buttonClass('secondary', 'sm')}>
            <ChevronLeft aria-hidden="true" />
            Previous
          </button>
          <span className="tabular text-[14px] text-ink-muted">
            Page {data.page} of {data.totalPages} · {data.totalCount} transactions
          </span>
          <button type="button" disabled={!data.hasNext} onClick={() => setFilter('page', data.page + 1)} className={buttonClass('secondary', 'sm')}>
            Next
            <ChevronRight aria-hidden="true" />
          </button>
        </nav>
      )}
    </div>
  )
}

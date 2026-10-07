import { useSearchParams, Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Download, Upload } from 'lucide-react'
import { useCategories } from '../../hooks/useCategories'
import { useTransactions } from '../../hooks/useTransactions'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { AddTransactionForm } from '../../components/AddTransactionForm'
import { MonthPicker } from '../../components/MonthPicker'
import { EmptyState } from '../../components/ui/EmptyState'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { downloadTransactionsExport } from '../../lib/api/import'
import { buttonClass, cardClass, cardTitleClass, pageTitleClass } from '../../lib/formStyles'
import { formatDayLabel } from '../../lib/format'
import { useToast } from '../../lib/toast/ToastContext'
import type { TransactionFilters } from '../../lib/api/types'
import { FilterBar } from './FilterBar'
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
    return <PageSkeleton label="Loading transactions…" shape="list" />
  }

  // Newest-first lists read as a diary, one group per day; an amount sort has no days to group by.
  const grouped = !filters.sort
  const groups: { day: string; rows: typeof data.transactions }[] = grouped ? [] : [{ day: '', rows: data.transactions }]
  if (grouped) {
    for (const t of data.transactions) {
      const last = groups[groups.length - 1]
      if (last?.day === t.occurredOn) last.rows.push(t)
      else groups.push({ day: t.occurredOn, rows: [t] })
    }
  }

  const list =
    data.transactions.length === 0 ? (
      <div className="rounded-tile border border-border bg-surface">
        {hasFilters ? (
          <EmptyState
            art="search"
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
          <EmptyState
            art="receipt"
            title={data.allMonths ? 'No transactions yet' : `No transactions in ${data.monthLabel}`}
            actions={
              <Link to="/transactions/import" className={buttonClass('secondary')}>
                <Upload aria-hidden="true" />
                Import a CSV
              </Link>
            }
          >
            {isDesktop ? 'Add one with the form beside the list' : 'Tap the + button below to add one'}, or bring in a CSV from your bank.
          </EmptyState>
        )}
      </div>
    ) : (
      <div className="space-y-5">
        {groups.map((group) => (
          <section key={group.day || 'all'} aria-labelledby={group.day ? `day-${group.day}` : undefined} aria-label={group.day ? undefined : 'Transaction list'}>
            {group.day && (
              <h2 id={`day-${group.day}`} className="mb-2 px-1 text-[13px] leading-[18px] font-semibold text-ink-muted">
                {formatDayLabel(group.day, data.allMonths)}
              </h2>
            )}
            <ul className="divide-y divide-border overflow-hidden rounded-tile border border-border bg-surface">
              {group.rows.map((t) => (
                <TransactionRow key={t.id} transaction={t} showDate={!grouped} showYear={data.allMonths} categories={allCategories} />
              ))}
            </ul>
          </section>
        ))}
      </div>
    )

  return (
    <div className="space-y-5 md:space-y-7">
      <div className="flex flex-wrap items-end justify-between gap-x-4 gap-y-3">
        <div>
          <h1 className={pageTitleClass}>Transactions</h1>
          <p className="tabular mt-1 text-[14px] leading-5 text-ink-muted">
            {data.totalCount} {data.totalCount === 1 ? 'transaction' : 'transactions'} {data.allMonths ? 'in all' : `in ${data.monthLabel}`}
          </p>
        </div>
        <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
          <MonthPicker
            value={data.monthValue}
            label={data.monthLabel}
            currentMonthValue={data.currentMonthValue}
            availableMonths={data.availableMonths}
            onChange={(v) => setFilter('month', v)}
            allowAllMonths
          />
          <div className="ml-auto flex gap-2 sm:ml-0">
            <Link to="/transactions/import" aria-label="Import CSV" className={`${buttonClass('secondary')} max-sm:w-11 max-sm:px-0`}>
              <Upload aria-hidden="true" />
              <span className="max-sm:hidden">Import</span>
            </Link>
            <button type="button" aria-label="Export CSV" onClick={() => void onExport()} className={`${buttonClass('secondary')} max-sm:w-11 max-sm:px-0`}>
              <Download aria-hidden="true" />
              <span className="max-sm:hidden">Export</span>
            </button>
          </div>
        </div>
      </div>

      <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-7">
        {isDesktop && (
          <section aria-labelledby="add-title" className={`${cardClass} lg:sticky lg:top-24 lg:order-last`}>
            <h2 id="add-title" className={`${cardTitleClass} mb-5`}>
              New transaction
            </h2>
            <AddTransactionForm categories={allCategories} />
          </section>
        )}

        <div className="min-w-0 space-y-5">
          <FilterBar filters={filters} categories={allCategories} onChange={setFilter} onClear={clearFilters} />
          {list}
          {data.totalPages > 1 && (
            <nav aria-label="Pages" className="flex items-center justify-between gap-3">
              <button type="button" disabled={!data.hasPrev} onClick={() => setFilter('page', data.page - 1)} className={buttonClass('secondary')}>
                <ChevronLeft aria-hidden="true" />
                Previous
              </button>
              <span className="tabular text-center text-[14px] leading-5 text-ink-muted">
                Page {data.page} of {data.totalPages}
              </span>
              <button type="button" disabled={!data.hasNext} onClick={() => setFilter('page', data.page + 1)} className={buttonClass('secondary')}>
                Next
                <ChevronRight aria-hidden="true" />
              </button>
            </nav>
          )}
        </div>
      </div>
    </div>
  )
}

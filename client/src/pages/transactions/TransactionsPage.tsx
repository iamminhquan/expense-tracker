import { useSearchParams, Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Inbox, SearchX } from 'lucide-react'
import { useCategories } from '../../hooks/useCategories'
import { useTransactions } from '../../hooks/useTransactions'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { MonthPicker } from '../../components/MonthPicker'
import { EmptyState } from '../../components/ui/EmptyState'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { buttonClass, listCardClass, pageTitleClass } from '../../lib/formStyles'
import { formatDayLabel } from '../../lib/format'
import type { Transaction, TransactionFilters } from '../../lib/api/types'
import { AddTransactionForm } from '../../components/AddTransactionForm'
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

interface DayGroup {
  date: string
  transactions: Transaction[]
}

// The rows arrive newest first, so a day's rows are already next to each other.
function groupByDay(transactions: Transaction[]): DayGroup[] {
  const groups: DayGroup[] = []
  for (const t of transactions) {
    const last = groups[groups.length - 1]
    if (last && last.date === t.occurredOn) last.transactions.push(t)
    else groups.push({ date: t.occurredOn, transactions: [t] })
  }
  return groups
}

export function TransactionsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = filtersFromSearchParams(searchParams)
  const { data, error, refetch } = useTransactions(filters)
  const { data: categories } = useCategories()
  const isDesktop = useIsDesktop()
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

  if (!data) {
    if (error) return <InlineError message="Could not load transactions." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading transactions…" />
  }

  const grouped = !isDesktop && !filters.sort
  const rows = data.transactions

  return (
    <div className="space-y-4 md:space-y-5">
      <div className="flex items-center justify-between gap-3">
        <h1 className={pageTitleClass}>Transactions</h1>
        <MonthPicker
          value={data.monthValue}
          label={data.monthLabel}
          currentMonthValue={data.currentMonthValue}
          availableMonths={data.availableMonths}
          onChange={(v) => setFilter('month', v)}
          allowAllMonths
          compact
        />
      </div>

      {isDesktop && <AddTransactionForm categories={allCategories} />}

      <FilterBar filters={filters} categories={allCategories} onChange={setFilter} onClear={clearFilters} />

      <section aria-label="Transaction list">
        {rows.length === 0 ? (
          <div className={listCardClass}>
            {hasFilters ? (
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
                {isDesktop ? 'Add one with the form above, or ' : 'Tap + to add one, or '}
                <Link to="/transactions/import" className="font-semibold text-accent-text underline underline-offset-2">
                  import a CSV
                </Link>{' '}
                from your bank.
              </EmptyState>
            )}
          </div>
        ) : grouped ? (
          <div className="space-y-5">
            {groupByDay(rows).map((group) => (
              <div key={group.date}>
                <h2 className="mb-2 px-1 text-[13px] leading-[18px] font-semibold text-ink-muted">{formatDayLabel(group.date, data.allMonths)}</h2>
                <ul className={listCardClass}>
                  {group.transactions.map((t) => (
                    <TransactionRow key={t.id} transaction={t} showYear={data.allMonths} categories={allCategories} dateInHeading />
                  ))}
                </ul>
              </div>
            ))}
          </div>
        ) : (
          <div className={listCardClass}>
            <ul>
              {rows.map((t) => (
                <TransactionRow key={t.id} transaction={t} showYear={data.allMonths} categories={allCategories} />
              ))}
            </ul>
          </div>
        )}
      </section>

      {data.totalPages > 1 && (
        <nav aria-label="Pages" className="flex flex-wrap items-center justify-between gap-3">
          <button type="button" disabled={!data.hasPrev} onClick={() => setFilter('page', data.page - 1)} className={buttonClass('secondary', 'sm')}>
            <ChevronLeft aria-hidden="true" />
            Previous
          </button>
          <span className="tabular order-last w-full text-center text-[13px] text-ink-muted sm:order-none sm:w-auto sm:text-[14px]">
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

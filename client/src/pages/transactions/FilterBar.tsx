import { useState } from 'react'
import { Search, SlidersHorizontal, X } from 'lucide-react'
import { BottomSheet } from '../../components/BottomSheet'
import { AmountInput } from '../../components/ui/AmountInput'
import { Field } from '../../components/ui/Field'
import { SelectControl } from '../../components/ui/SelectControl'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { buttonClass, searchInputClass } from '../../lib/formStyles'
import { formatVND } from '../../lib/format'
import type { Category, TransactionFilters } from '../../lib/api/types'

interface FilterBarProps {
  filters: TransactionFilters
  categories: Category[]
  onChange: <K extends keyof TransactionFilters>(key: K, value: TransactionFilters[K]) => void
  onClear: () => void
}

function numberOrUndefined(value: string): number | undefined {
  return value === '' ? undefined : Number(value)
}

export function FilterBar({ filters, categories, onChange, onClear }: FilterBarProps) {
  const isDesktop = useIsDesktop()
  const [sheetOpen, setSheetOpen] = useState(false)

  const chips: { key: keyof TransactionFilters; label: string }[] = []
  if (filters.type) chips.push({ key: 'type', label: filters.type === 'expense' ? 'Expenses' : 'Income' })
  if (filters.category) {
    const category = categories.find((c) => c.id === filters.category)
    chips.push({ key: 'category', label: category?.name ?? 'Category' })
  }
  if (filters.min !== undefined) chips.push({ key: 'min', label: `From ${formatVND(filters.min)}` })
  if (filters.max !== undefined) chips.push({ key: 'max', label: `Up to ${formatVND(filters.max)}` })
  if (filters.sort) chips.push({ key: 'sort', label: filters.sort === 'amount_desc' ? 'Largest first' : 'Smallest first' })

  const typeSelect = (props: object) => (
    <SelectControl {...props} value={filters.type ?? ''} onChange={(e) => onChange('type', (e.target.value || undefined) as TransactionFilters['type'])}>
      <option value="">All types</option>
      <option value="expense">Expense</option>
      <option value="income">Income</option>
    </SelectControl>
  )
  const categorySelect = (props: object) => (
    <SelectControl {...props} value={filters.category ?? ''} onChange={(e) => onChange('category', e.target.value ? Number(e.target.value) : undefined)}>
      <option value="">All categories</option>
      {categories.map((c) => (
        <option key={c.id} value={c.id}>
          {c.name}
        </option>
      ))}
    </SelectControl>
  )
  const sortSelect = (props: object) => (
    <SelectControl {...props} value={filters.sort ?? ''} onChange={(e) => onChange('sort', (e.target.value || undefined) as TransactionFilters['sort'])}>
      <option value="">Newest first</option>
      <option value="amount_desc">Amount: high to low</option>
      <option value="amount_asc">Amount: low to high</option>
    </SelectControl>
  )
  const amountRange = (
    <div className="grid grid-cols-2 gap-2">
      <AmountInput aria-label="Minimum amount" placeholder="Min" value={filters.min ?? ''} onChange={(e) => onChange('min', numberOrUndefined(e.target.value))} />
      <AmountInput aria-label="Maximum amount" placeholder="Max" value={filters.max ?? ''} onChange={(e) => onChange('max', numberOrUndefined(e.target.value))} />
    </div>
  )

  return (
    <div role="search" aria-label="Filter transactions" className="space-y-3">
      <div className="flex gap-2">
        <div className="relative min-w-0 flex-1">
          <Search aria-hidden="true" className="pointer-events-none absolute top-1/2 left-4 size-[18px] -translate-y-1/2 text-ink-muted" />
          <input
            type="search"
            aria-label="Search transactions"
            placeholder="Search notes…"
            value={filters.q ?? ''}
            onChange={(e) => onChange('q', e.target.value || undefined)}
            className={searchInputClass}
          />
        </div>
        {!isDesktop && (
          <button type="button" aria-haspopup="dialog" onClick={() => setSheetOpen(true)} className={buttonClass('secondary')}>
            <SlidersHorizontal aria-hidden="true" />
            Filters
            {chips.length > 0 && (
              <span className="tabular flex size-6 items-center justify-center rounded-full bg-accent text-[12px] text-on-accent">
                {chips.length}
                <span className="sr-only"> active</span>
              </span>
            )}
          </button>
        )}
      </div>

      {isDesktop ? (
        <div className="grid grid-cols-[repeat(3,minmax(0,1fr))_minmax(0,1.25fr)] items-center gap-2">
          {typeSelect({ 'aria-label': 'Transaction type' })}
          {categorySelect({ 'aria-label': 'Category' })}
          {sortSelect({ 'aria-label': 'Sort order' })}
          {amountRange}
        </div>
      ) : (
        <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)} label="Filters">
          <h2 className="heading mb-4 text-[22px] leading-7 text-ink">Filters</h2>
          <div className="grid gap-4">
            <Field label="Type">{(control) => typeSelect(control)}</Field>
            <Field label="Category">{(control) => categorySelect(control)}</Field>
            <Field label="Sort">{(control) => sortSelect(control)}</Field>
            <div>
              <p aria-hidden="true" className="mb-2 text-[13px] leading-[18px] font-semibold text-ink">
                Amount
              </p>
              {amountRange}
            </div>
          </div>
          <div className="mt-6 grid grid-cols-2 gap-3">
            <button type="button" onClick={onClear} disabled={chips.length === 0} className={buttonClass('secondary', 'lg')}>
              Clear all
            </button>
            <button type="button" onClick={() => setSheetOpen(false)} className={buttonClass('primary', 'lg')}>
              Done
            </button>
          </div>
        </BottomSheet>
      )}

      {chips.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {chips.map((chip) => (
            <span key={chip.key} className="inline-flex h-9 items-center gap-0.5 rounded-full bg-accent-tint pr-1 pl-3.5 text-[13px] leading-4 font-semibold text-expense">
              <span className="tabular">{chip.label}</span>
              <button
                type="button"
                onClick={() => onChange(chip.key, undefined)}
                aria-label={`Remove filter: ${chip.label}`}
                className="relative flex size-7 items-center justify-center rounded-full before:absolute before:-inset-2 before:content-[''] hover:bg-surface/70"
              >
                <X aria-hidden="true" className="size-3.5" />
              </button>
            </span>
          ))}
          <button type="button" onClick={onClear} className="inline-flex min-h-11 items-center rounded-full px-3 text-[13px] font-semibold text-ink underline-offset-4 hover:underline">
            Clear all
          </button>
        </div>
      )}
    </div>
  )
}

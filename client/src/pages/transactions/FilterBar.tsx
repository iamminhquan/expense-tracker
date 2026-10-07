import { useId, useState, type ReactNode } from 'react'
import { Search, SlidersHorizontal, X } from 'lucide-react'
import { AmountInput } from '../../components/ui/AmountInput'
import { SelectControl } from '../../components/ui/SelectControl'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { buttonClass, inputClass } from '../../lib/formStyles'
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
  const [panelOpen, setPanelOpen] = useState(false)
  const panelId = useId()

  const chips: { key: keyof TransactionFilters; label: string }[] = []
  if (filters.type) chips.push({ key: 'type', label: filters.type === 'expense' ? 'Expenses' : 'Income' })
  if (filters.category) {
    const category = categories.find((c) => c.id === filters.category)
    chips.push({ key: 'category', label: category?.name ?? 'Category' })
  }
  if (filters.min !== undefined) chips.push({ key: 'min', label: `From ${formatVND(filters.min)}` })
  if (filters.max !== undefined) chips.push({ key: 'max', label: `Up to ${formatVND(filters.max)}` })
  if (filters.sort) chips.push({ key: 'sort', label: filters.sort === 'amount_desc' ? 'Largest first' : 'Smallest first' })

  const controls = (
    <>
      <SelectControl
        aria-label="Transaction type"
        value={filters.type ?? ''}
        onChange={(e) => onChange('type', (e.target.value || undefined) as TransactionFilters['type'])}
      >
        <option value="">All types</option>
        <option value="expense">Expense</option>
        <option value="income">Income</option>
      </SelectControl>
      <SelectControl
        aria-label="Category"
        value={filters.category ?? ''}
        onChange={(e) => onChange('category', e.target.value ? Number(e.target.value) : undefined)}
      >
        <option value="">All categories</option>
        {categories.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </SelectControl>
      <SelectControl
        aria-label="Sort order"
        value={filters.sort ?? ''}
        onChange={(e) => onChange('sort', (e.target.value || undefined) as TransactionFilters['sort'])}
      >
        <option value="">Newest first</option>
        <option value="amount_desc">Amount: high to low</option>
        <option value="amount_asc">Amount: low to high</option>
      </SelectControl>
      <div className="grid grid-cols-2 gap-2">
        <AmountInput
          aria-label="Minimum amount"
          placeholder="Min"
          value={filters.min ?? ''}
          onChange={(e) => onChange('min', numberOrUndefined(e.target.value))}
          className="text-[15px]"
        />
        <AmountInput
          aria-label="Maximum amount"
          placeholder="Max"
          value={filters.max ?? ''}
          onChange={(e) => onChange('max', numberOrUndefined(e.target.value))}
          className="text-[15px]"
        />
      </div>
    </>
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
            className={`${inputClass} rounded-full pl-[44px] shadow-card`}
          />
        </div>
        {!isDesktop && (
          <button
            type="button"
            aria-expanded={panelOpen}
            aria-controls={panelId}
            onClick={() => setPanelOpen((o) => !o)}
            className={`${buttonClass(panelOpen || chips.length > 0 ? 'tonal' : 'secondary')} rounded-full`}
          >
            <SlidersHorizontal aria-hidden="true" />
            Filters
            {chips.length > 0 && (
              <span className="tabular flex size-6 items-center justify-center rounded-full bg-accent text-[12px] text-on-accent">{chips.length}</span>
            )}
          </button>
        )}
      </div>

      {isDesktop ? (
        <div className="grid grid-cols-[repeat(3,minmax(0,1fr))_minmax(0,1.3fr)] items-center gap-2">{controls}</div>
      ) : (
        panelOpen && (
          <div id={panelId} className="grid animate-pop-in gap-2.5 rounded-card border border-border bg-surface p-3 shadow-card">
            {controls}
          </div>
        )
      )}

      {chips.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {chips.map((chip) => (
            <Chip key={chip.key} onRemove={() => onChange(chip.key, undefined)} label={chip.label}>
              {chip.label}
            </Chip>
          ))}
          <button type="button" onClick={onClear} className="relative h-8 rounded-full px-3 text-[13px] font-semibold text-accent-text underline-offset-2 before:absolute before:-inset-y-1.5 before:inset-x-0 before:content-[''] hover:underline">
            Clear all
          </button>
        </div>
      )}
    </div>
  )
}

function Chip({ label, onRemove, children }: { label: string; onRemove: () => void; children: ReactNode }) {
  return (
    <span className="inline-flex h-8 items-center gap-1 rounded-full bg-accent-tint pr-1.5 pl-3 text-[13px] leading-4 font-semibold text-accent-text">
      <span className="tabular">{children}</span>
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove filter: ${label}`}
        className="relative flex size-6 items-center justify-center rounded-full before:absolute before:-inset-2.5 before:content-[''] hover:bg-surface"
      >
        <X aria-hidden="true" className="size-3.5" />
      </button>
    </span>
  )
}

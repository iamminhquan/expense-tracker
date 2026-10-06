import type { Category, TransactionFilters } from '../../lib/api/types'

interface FilterBarProps {
  filters: TransactionFilters
  categories: Category[]
  onChange: <K extends keyof TransactionFilters>(key: K, value: TransactionFilters[K]) => void
}

export function FilterBar({ filters, categories, onChange }: FilterBarProps) {
  return (
    <div className="flex flex-wrap gap-2">
      <input
        type="search"
        aria-label="Search transactions"
        placeholder="Search…"
        value={filters.q ?? ''}
        onChange={(e) => onChange('q', e.target.value || undefined)}
        className="min-w-[140px] flex-1 rounded-[10px] border border-border-input bg-surface px-3 py-1.5 text-[13px]"
      />
      <select
        aria-label="Transaction type"
        value={filters.type ?? ''}
        onChange={(e) => onChange('type', (e.target.value || undefined) as TransactionFilters['type'])}
        className="rounded-[10px] border border-border-input bg-surface px-2 py-1.5 text-[13px]"
      >
        <option value="">All types</option>
        <option value="expense">Expense</option>
        <option value="income">Income</option>
      </select>
      <select
        aria-label="Category"
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
        aria-label="Sort order"
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

import { useCategories } from '../../hooks/useCategories'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { cardTitleClass, pageTitleClass } from '../../lib/formStyles'
import type { Category } from '../../lib/api/types'
import { AddCategoryForm } from './AddCategoryForm'
import { CategoryRow } from './CategoryRow'

export function CategoriesPage() {
  const { data, error, refetch } = useCategories()

  if (!data) {
    if (error) return <InlineError message="Could not load categories." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading categories…" shape="cards" />
  }

  return (
    <div className="space-y-5 md:space-y-7">
      <div>
        <h1 className={pageTitleClass}>Categories</h1>
        <p className="mt-1 text-[14px] leading-5 text-ink-muted">Each color follows its category into every list and chart.</p>
      </div>
      <div className="grid items-start gap-5 md:grid-cols-2 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_340px] lg:gap-6">
        <CategoryGroup title="Expense" categories={data.expenseCategories} />
        <CategoryGroup title="Income" categories={data.incomeCategories} />
        <div className="md:col-span-2 lg:sticky lg:top-24 lg:col-span-1">
          <AddCategoryForm />
        </div>
      </div>
    </div>
  )
}

function CategoryGroup({ title, categories }: { title: string; categories: Category[] }) {
  const id = `group-${title.toLowerCase()}`
  return (
    <section aria-labelledby={id}>
      <h2 id={id} className="mb-2 flex items-baseline gap-2 px-1">
        <span className={cardTitleClass}>{title}</span>
        <span className="tabular text-[13px] font-medium text-ink-muted">{categories.length}</span>
      </h2>
      <ul className="divide-y divide-border overflow-hidden rounded-tile border border-border bg-surface">
        {categories.map((c) => (
          <CategoryRow key={c.id} category={c} />
        ))}
      </ul>
    </section>
  )
}

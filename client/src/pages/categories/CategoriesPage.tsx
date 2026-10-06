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
    return <PageSkeleton label="Loading categories…" />
  }

  return (
    <div className="space-y-4 md:space-y-6">
      <h1 className={pageTitleClass}>Categories</h1>
      <div className="grid items-start gap-4 md:gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_360px]">
        <CategoryGroup title="Expense" categories={data.expenseCategories} />
        <CategoryGroup title="Income" categories={data.incomeCategories} />
        <div className="lg:sticky lg:top-24">
          <AddCategoryForm />
        </div>
      </div>
    </div>
  )
}

function CategoryGroup({ title, categories }: { title: string; categories: Category[] }) {
  const id = `group-${title.toLowerCase()}`
  return (
    <section aria-labelledby={id} className="overflow-hidden rounded-[24px] border border-border bg-surface md:rounded-[28px]">
      <h2 id={id} className={`${cardTitleClass} px-4 pt-5 pb-3 md:px-5`}>
        {title} <span className="tabular font-sans text-[14px] font-medium text-ink-muted">· {categories.length}</span>
      </h2>
      <ul className="border-t border-border">
        {categories.map((c) => (
          <CategoryRow key={c.id} category={c} />
        ))}
      </ul>
    </section>
  )
}

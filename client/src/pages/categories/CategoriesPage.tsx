import { useState } from 'react'
import { Plus, Tags, X } from 'lucide-react'
import { useCategories } from '../../hooks/useCategories'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { EmptyState } from '../../components/ui/EmptyState'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { buttonClass, cardTitleClass, listCardClass, pageTitleClass } from '../../lib/formStyles'
import type { Category } from '../../lib/api/types'
import { AddCategoryForm } from './AddCategoryForm'
import { CategoryRow } from './CategoryRow'

export function CategoriesPage() {
  const { data, error, refetch } = useCategories()
  const isDesktop = useIsDesktop()
  const [formOpen, setFormOpen] = useState(false)

  if (!data) {
    if (error) return <InlineError message="Could not load categories." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading categories…" />
  }

  return (
    <div className="space-y-4 md:space-y-5">
      <div className="flex items-center justify-between gap-3">
        <h1 className={pageTitleClass}>Categories</h1>
        {!isDesktop && (
          <button
            type="button"
            aria-expanded={formOpen}
            onClick={() => setFormOpen((o) => !o)}
            className={`${buttonClass(formOpen ? 'secondary' : 'tonal')} rounded-full`}
          >
            {formOpen ? <X aria-hidden="true" /> : <Plus aria-hidden="true" />}
            {formOpen ? 'Close' : 'New'}
          </button>
        )}
      </div>
      {!isDesktop && formOpen && (
        <div className="animate-pop-in">
          <AddCategoryForm onAdded={() => setFormOpen(false)} />
        </div>
      )}
      <div className="grid items-start gap-4 md:gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_360px]">
        <CategoryGroup title="Expense" categories={data.expenseCategories} />
        <CategoryGroup title="Income" categories={data.incomeCategories} />
        {isDesktop && (
          <div className="sticky top-24">
            <AddCategoryForm />
          </div>
        )}
      </div>
    </div>
  )
}

function CategoryGroup({ title, categories }: { title: string; categories: Category[] }) {
  const id = `group-${title.toLowerCase()}`
  return (
    <section aria-labelledby={id} className={listCardClass}>
      <h2 id={id} className={`${cardTitleClass} flex items-center gap-2 px-4 pt-5 pb-3.5 md:px-5`}>
        {title}
        <span className="tabular rounded-full bg-surface-2 px-2 py-0.5 font-sans text-[12px] leading-4 font-semibold text-ink-muted">{categories.length}</span>
      </h2>
      {categories.length === 0 ? (
        <div className="border-t border-border">
          <EmptyState icon={<Tags />} title={`No ${title.toLowerCase()} categories`}>
            Create one and it shows up here.
          </EmptyState>
        </div>
      ) : (
        <ul className="border-t border-border">
          {categories.map((c) => (
            <CategoryRow key={c.id} category={c} />
          ))}
        </ul>
      )}
    </section>
  )
}

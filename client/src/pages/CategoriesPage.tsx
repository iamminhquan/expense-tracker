import { useState } from 'react'
import { useCategories, useCreateCategory, useDeleteCategory, useUpdateCategory } from '../hooks/useCategories'
import { ApiError } from '../lib/api/client'
import type { Category } from '../lib/api/types'

const SWATCHES = ['#D97757', '#5B8DEF', '#8B7BD8', '#6BA292', '#E0A82E', '#D97AA0', '#4FA871', '#7CA65C']

export function CategoriesPage() {
  const { data, isLoading } = useCategories()
  const createCategory = useCreateCategory()
  const [type, setType] = useState<'expense' | 'income'>('expense')
  const [name, setName] = useState('')
  const [color, setColor] = useState(SWATCHES[0])
  const [error, setError] = useState<string | null>(null)

  if (isLoading || !data) return <p className="text-ink-faint">Loading…</p>

  async function onCreate(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await createCategory.mutateAsync({ name, type, color })
      setName('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create the category.')
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-[20px] font-semibold">Categories</h1>

      <form onSubmit={onCreate} className="rounded-[16px] border border-border-card bg-surface p-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex gap-1 rounded-[9px] bg-track p-[3px]">
            {(['expense', 'income'] as const).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => setType(t)}
                className={`rounded-[6px] px-3 py-1.5 text-[13px] capitalize ${
                  type === t ? 'bg-surface font-semibold text-ink shadow-sm' : 'text-ink-faint'
                }`}
              >
                {t}
              </button>
            ))}
          </div>
          <input
            placeholder="Category name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="min-w-[160px] flex-1 rounded-[10px] border border-border-input bg-surface px-3 py-2 text-[14px]"
          />
          <div className="flex gap-1.5">
            {SWATCHES.map((s) => (
              <button
                key={s}
                type="button"
                aria-label={`Color ${s}`}
                onClick={() => setColor(s)}
                className="h-6 w-6 rounded-full"
                style={{ backgroundColor: s, outline: color === s ? '2px solid rgb(var(--c-ink))' : undefined, outlineOffset: 2 }}
              />
            ))}
          </div>
          <button
            type="submit"
            disabled={createCategory.isPending}
            className="rounded-[10px] bg-accent px-4 py-2 text-[13px] font-semibold text-on-solid hover:opacity-90 disabled:opacity-50"
          >
            Add
          </button>
        </div>
        {error && <p className="mt-2 text-[13px] text-expense">{error}</p>}
      </form>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <CategoryList title="Expense" categories={data.expenseCategories} />
        <CategoryList title="Income" categories={data.incomeCategories} />
      </div>
    </div>
  )
}

function CategoryList({ title, categories }: { title: string; categories: Category[] }) {
  return (
    <div className="rounded-[16px] border border-border-card bg-surface p-4">
      <p className="mb-3 text-[13px] font-semibold uppercase tracking-wide text-ink-faint">{title}</p>
      {categories.length === 0 ? (
        <p className="text-[13px] text-ink-faint">No categories yet.</p>
      ) : (
        <ul className="divide-y divide-border-list">
          {categories.map((c) => (
            <CategoryRow key={c.id} category={c} />
          ))}
        </ul>
      )}
    </div>
  )
}

function CategoryRow({ category }: { category: Category }) {
  const updateCategory = useUpdateCategory()
  const deleteCategory = useDeleteCategory()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(category.name)
  const [error, setError] = useState<string | null>(null)

  async function onSave(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateCategory.mutateAsync({ id: category.id, input: { name } })
      setEditing(false)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not rename the category.')
    }
  }

  async function onDelete() {
    if (!confirm(`Delete "${category.name}"? Its transactions move to Other.`)) return
    try {
      await deleteCategory.mutateAsync(category.id)
    } catch (err) {
      alert(err instanceof ApiError ? err.message : 'Could not delete the category.')
    }
  }

  if (editing) {
    return (
      <li className="py-2">
        <form onSubmit={onSave} className="flex items-center gap-2">
          <span className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: category.color }} />
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="flex-1 rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          />
          <button type="submit" className="text-[12px] font-semibold text-accent">
            Save
          </button>
          <button type="button" onClick={() => setEditing(false)} className="text-[12px] text-ink-faint">
            Cancel
          </button>
        </form>
        {error && <p className="mt-1 text-[12px] text-expense">{error}</p>}
      </li>
    )
  }

  return (
    <li className="flex items-center gap-2 py-2 text-[13px]">
      <span className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: category.color }} />
      <span className="flex-1 text-ink">{category.name}</span>
      <span className="text-ink-faint">{category.transactionCount}</span>
      {!category.isDefault && (
        <>
          <button onClick={() => setEditing(true)} className="text-ink-faint hover:text-ink" aria-label="Rename">
            Edit
          </button>
          <button onClick={() => void onDelete()} className="text-ink-faint hover:text-expense" aria-label="Delete">
            Delete
          </button>
        </>
      )}
    </li>
  )
}

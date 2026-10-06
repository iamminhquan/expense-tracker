import { useState, type FormEvent } from 'react'
import { ArrowDownLeft, ArrowUpRight, Plus } from 'lucide-react'
import { useCreateCategory } from '../../hooks/useCategories'
import { ApiError } from '../../lib/api/client'
import { SWATCHES } from '../../lib/categorySwatches'
import { buttonClass, cardClass, cardTitleClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Field } from '../../components/ui/Field'
import { SegmentedControl } from '../../components/ui/SegmentedControl'
import type { Category } from '../../lib/api/types'
import { SwatchPicker } from './SwatchPicker'

export function AddCategoryForm() {
  const createCategory = useCreateCategory()
  const toast = useToast()
  const [type, setType] = useState<Category['type']>('expense')
  const [name, setName] = useState('')
  const [color, setColor] = useState(SWATCHES[0])
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await createCategory.mutateAsync({ name: name.trim(), type, color })
      toast.success(`Added "${name.trim()}"`)
      setName('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the category.')
    }
  }

  return (
    <section aria-labelledby="new-category-title" className={cardClass}>
      <h2 id="new-category-title" className={`${cardTitleClass} mb-5`}>
        New category
      </h2>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <span aria-hidden="true" className="mb-1.5 block text-[13px] leading-[18px] font-semibold text-ink">
            Type
          </span>
          <SegmentedControl
            label="Category type"
            value={type}
            onChange={setType}
            size="tall"
            fullWidth
            options={[
              { value: 'expense', label: 'Expense', icon: <ArrowUpRight aria-hidden="true" /> },
              { value: 'income', label: 'Income', icon: <ArrowDownLeft aria-hidden="true" /> },
            ]}
          />
        </div>
        <Field label="Name" error={error}>
          {(control) => (
            <input {...control} required maxLength={40} value={name} onChange={(e) => setName(e.target.value)} placeholder="Cà phê" className={inputClass} />
          )}
        </Field>
        <div>
          <span aria-hidden="true" className="mb-1.5 block text-[13px] leading-[18px] font-semibold text-ink">
            Color
          </span>
          <SwatchPicker value={color} onChange={setColor} />
        </div>
        <button type="submit" disabled={createCategory.isPending} aria-busy={createCategory.isPending} className={`${buttonClass('primary')} w-full`}>
          <Plus aria-hidden="true" />
          Add category
        </button>
      </form>
    </section>
  )
}

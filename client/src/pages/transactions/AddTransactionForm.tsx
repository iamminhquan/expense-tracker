import { useState, type FormEvent } from 'react'
import { ArrowDownLeft, ArrowUpRight, Plus } from 'lucide-react'
import { useCreateTransaction } from '../../hooks/useTransactions'
import { ApiError } from '../../lib/api/client'
import { buttonClass, cardClass, cardTitleClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { useMediaQuery } from '../../hooks/useMediaQuery'
import { AmountInput } from '../../components/ui/AmountInput'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { SegmentedControl } from '../../components/ui/SegmentedControl'
import { SelectControl } from '../../components/ui/SelectControl'
import type { Category, Transaction } from '../../lib/api/types'

// Local date, not toISOString(): UTC would still be yesterday before 7am in Vietnam.
function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export function AddTransactionForm({ categories }: { categories: Category[] }) {
  const createTransaction = useCreateTransaction()
  const toast = useToast()
  const [type, setType] = useState<Transaction['type']>('expense')
  const [categoryId, setCategoryId] = useState<number | ''>('')
  const [amount, setAmount] = useState('')
  const [occurredOn, setOccurredOn] = useState(todayISO())
  const [description, setDescription] = useState('')
  const [categoryError, setCategoryError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const isWide = useMediaQuery('(min-width: 1024px)')
  const options = categories.filter((c) => c.type === type)

  function changeType(next: Transaction['type']) {
    setType(next)
    setCategoryId('')
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!categoryId) {
      setCategoryError('Choose a category.')
      return
    }
    try {
      await createTransaction.mutateAsync({ categoryId: Number(categoryId), amount: Number(amount), occurredOn, description, type })
      setAmount('')
      setDescription('')
      toast.success(type === 'expense' ? 'Expense added' : 'Income added')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the transaction.')
    }
  }

  return (
    <section aria-labelledby="add-title" className={cardClass}>
      <h2 id="add-title" className={`${cardTitleClass} mb-5`}>
        Add a transaction
      </h2>
      <form onSubmit={onSubmit}>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[auto_minmax(0,1fr)_minmax(0,0.9fr)_auto_minmax(0,1.3fr)_auto] lg:items-start">
          <div>
            <span className="mb-1.5 block text-[13px] leading-[18px] font-semibold text-ink" aria-hidden="true">
              Type
            </span>
            <SegmentedControl
              label="Transaction type"
              value={type}
              onChange={changeType}
              size={isWide ? 'md' : 'tall'}
              fullWidth={!isWide}
              options={[
                { value: 'expense', label: 'Expense', icon: <ArrowUpRight aria-hidden="true" /> },
                { value: 'income', label: 'Income', icon: <ArrowDownLeft aria-hidden="true" /> },
              ]}
            />
          </div>
          <Field label="Category" error={categoryError}>
            {(control) => (
              <SelectControl
                {...control}
                value={categoryId}
                onChange={(e) => {
                  setCategoryId(Number(e.target.value))
                  setCategoryError(null)
                }}
              >
                <option value="" disabled>
                  Choose…
                </option>
                {options.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </SelectControl>
            )}
          </Field>
          <Field label="Amount">
            {(control) => <AmountInput {...control} required value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0" />}
          </Field>
          <Field label="Date">
            {(control) => (
              <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />
            )}
          </Field>
          <Field label="Note (optional)">
            {(control) => (
              <input
                {...control}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Cà phê với Minh"
                className={inputClass}
              />
            )}
          </Field>
          <div className="lg:pt-[24px]">
            <button type="submit" disabled={createTransaction.isPending} aria-busy={createTransaction.isPending} className={`${buttonClass('primary')} w-full`}>
              <Plus aria-hidden="true" />
              Add
            </button>
          </div>
        </div>
        {error && (
          <div role="alert">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
      </form>
    </section>
  )
}

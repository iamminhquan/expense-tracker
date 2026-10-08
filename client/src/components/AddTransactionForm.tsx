import { useState, type FormEvent, type Ref } from 'react'
import { ArrowDownLeft, ArrowUpRight, Check, Plus } from 'lucide-react'
import { useCreateTransaction } from '../hooks/useTransactions'
import { useIsDesktop } from '../hooks/useMediaQuery'
import { ApiError } from '../lib/api/client'
import { buttonClass, cardClass, cardTitleClass, inputClass } from '../lib/formStyles'
import { useToast } from '../lib/toast/ToastContext'
import { AmountInput } from './ui/AmountInput'
import { Field } from './ui/Field'
import { FieldErrorText } from './ui/FieldErrorText'
import { SegmentedControl } from './ui/SegmentedControl'
import { SelectControl } from './ui/SelectControl'
import type { Category, Transaction } from '../lib/api/types'

interface AddTransactionFormProps {
  categories: Category[]
  /** Called after a transaction is saved, so the mobile sheet can close itself. */
  onAdded?: () => void
  amountRef?: Ref<HTMLInputElement>
}

// Local date, not toISOString(): UTC would still be yesterday before 7am in Vietnam.
function todayISO(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const TYPE_OPTIONS = [
  { value: 'expense', label: 'Expense', tone: 'expense', icon: <ArrowUpRight aria-hidden="true" /> },
  { value: 'income', label: 'Income', tone: 'income', icon: <ArrowDownLeft aria-hidden="true" /> },
] as const

export function AddTransactionForm({ categories, onAdded, amountRef }: AddTransactionFormProps) {
  const createTransaction = useCreateTransaction()
  const toast = useToast()
  const isDesktop = useIsDesktop()
  const [type, setType] = useState<Transaction['type']>('expense')
  const [categoryId, setCategoryId] = useState<number | ''>('')
  const [amount, setAmount] = useState('')
  const [occurredOn, setOccurredOn] = useState(todayISO())
  const [description, setDescription] = useState('')
  const [categoryError, setCategoryError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const options = categories.filter((c) => c.type === type)

  function changeType(next: Transaction['type']) {
    setType(next)
    setCategoryId('')
    setCategoryError(null)
  }

  function pickCategory(id: number) {
    setCategoryId(id)
    setCategoryError(null)
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
      onAdded?.()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the transaction.')
    }
  }

  const typeToggle = (
    <SegmentedControl
      label="Transaction type"
      value={type}
      onChange={changeType}
      size={isDesktop ? 'md' : 'tall'}
      fullWidth={!isDesktop}
      options={[...TYPE_OPTIONS]}
    />
  )
  const formError = error && (
    <div role="alert">
      <FieldErrorText>{error}</FieldErrorText>
    </div>
  )

  if (!isDesktop) {
    return (
      <form onSubmit={onSubmit} aria-labelledby="quick-add-title" className="space-y-5 pt-1">
        <h2 id="quick-add-title" className="font-display text-[26px] leading-8 font-bold tracking-[-0.03em] text-ink">
          Add transaction
        </h2>
        {typeToggle}
        <Field label="Amount">
          {(control) => <AmountInput {...control} ref={amountRef} large required value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0" />}
        </Field>
        <CategoryChips categories={options} value={categoryId} onChange={pickCategory} error={categoryError} />
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)] gap-3">
          <Field label="Date">
            {(control) => <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />}
          </Field>
          <Field label="Note (optional)">
            {(control) => <input {...control} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Cà phê với Minh" className={inputClass} />}
          </Field>
        </div>
        {formError}
        <div className="sticky bottom-0 -mx-5 bg-surface/95 px-5 pt-2 pb-1 backdrop-blur-sm">
          <button
            type="submit"
            disabled={createTransaction.isPending}
            aria-busy={createTransaction.isPending}
            className={`${buttonClass('primary')} h-14 w-full rounded-card text-[16px]`}
          >
            <Plus aria-hidden="true" />
            {type === 'expense' ? 'Add expense' : 'Add income'}
          </button>
        </div>
      </form>
    )
  }

  return (
    <section aria-labelledby="add-title" className={cardClass}>
      <form onSubmit={onSubmit}>
        <div className="mb-5 flex items-center justify-between gap-4">
          <h2 id="add-title" className={cardTitleClass}>
            Add a transaction
          </h2>
          {typeToggle}
        </div>
        {/* Firefox spells the date "10 / 09 / 2026" and clips the year below about 10.5rem. */}
        <div className="grid grid-cols-2 items-start gap-x-3 gap-y-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(10.5rem,0.8fr)_minmax(0,1.5fr)_auto]">
          <Field label="Amount">
            {(control) => <AmountInput {...control} ref={amountRef} required value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0" />}
          </Field>
          <Field label="Category" error={categoryError}>
            {(control) => (
              <SelectControl {...control} value={categoryId} onChange={(e) => pickCategory(Number(e.target.value))}>
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
          <Field label="Date">
            {(control) => <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={inputClass} />}
          </Field>
          <Field label="Note (optional)">
            {(control) => <input {...control} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Cà phê với Minh" className={inputClass} />}
          </Field>
          <div className="col-span-2 flex justify-end lg:col-span-1 lg:pt-[24px]">
            <button
              type="submit"
              disabled={createTransaction.isPending}
              aria-busy={createTransaction.isPending}
              className={`${buttonClass('primary')} w-40`}
            >
              <Plus aria-hidden="true" />
              {type === 'expense' ? 'Add expense' : 'Add income'}
            </button>
          </div>
        </div>
        {formError}
      </form>
    </section>
  )
}

interface CategoryChipsProps {
  categories: Category[]
  value: number | ''
  onChange: (id: number) => void
  error: string | null
}

function CategoryChips({ categories, value, onChange, error }: CategoryChipsProps) {
  return (
    <div>
      <span id="chips-label" className="mb-2 block text-[13px] leading-[18px] font-semibold text-ink">
        Category
      </span>
      <div role="radiogroup" aria-labelledby="chips-label" aria-describedby={error ? 'chips-error' : undefined} className="flex flex-wrap gap-2">
        {categories.map((c) => {
          const selected = c.id === value
          return (
            <label
              key={c.id}
              className={`relative flex h-11 cursor-pointer items-center gap-2 rounded-full border pr-4 pl-3 text-[14px] leading-5 font-semibold transition-[background-color,border-color,transform] duration-150 active:scale-95 motion-reduce:active:scale-100 ${
                selected ? 'border-accent-text bg-accent-tint text-accent-text' : 'border-border bg-surface text-ink hover:bg-surface-2'
              }`}
            >
              <input type="radio" name="quick-category" value={c.id} checked={selected} onChange={() => onChange(c.id)} className="peer sr-only" />
              <span aria-hidden="true" className="flex size-5 items-center justify-center rounded-full" style={{ backgroundColor: c.color }}>
                {selected && <Check strokeWidth={3.5} className="size-3 text-on-swatch" />}
              </span>
              {c.name}
              <span aria-hidden="true" className="pointer-events-none absolute -inset-0.5 rounded-full peer-focus-visible:outline-3 peer-focus-visible:outline-offset-1 peer-focus-visible:outline-accent-text" />
            </label>
          )
        })}
      </div>
      <FieldErrorText id="chips-error">{error}</FieldErrorText>
    </div>
  )
}

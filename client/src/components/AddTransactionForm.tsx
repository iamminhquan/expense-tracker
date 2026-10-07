import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { ArrowDownLeft, ArrowUpRight, CalendarDays, Plus } from 'lucide-react'
import { useCreateTransaction } from '../hooks/useTransactions'
import { ApiError } from '../lib/api/client'
import { buttonClass, chipClass, inputClass, labelClass } from '../lib/formStyles'
import { useToast } from '../lib/toast/ToastContext'
import { Field } from './ui/Field'
import { FieldErrorText } from './ui/FieldErrorText'
import { MoneyInput } from './ui/MoneyInput'
import { SegmentedControl } from './ui/SegmentedControl'
import type { Category, Transaction } from '../lib/api/types'

// Local dates, not toISOString(): UTC would still be yesterday before 7am in Vietnam.
function isoDaysAgo(days: number): string {
  const d = new Date()
  d.setDate(d.getDate() - days)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

interface AddTransactionFormProps {
  categories: Category[]
  /** Called after a successful add, e.g. to close the sheet the form sits in. */
  onAdded?: () => void
  /** The phone sheet opens straight onto the amount, so logging a spend is: open, type, tap. */
  autoFocusAmount?: boolean
}

export function AddTransactionForm({ categories, onAdded, autoFocusAmount }: AddTransactionFormProps) {
  const createTransaction = useCreateTransaction()
  const toast = useToast()
  const groupName = useId()
  const [type, setType] = useState<Transaction['type']>('expense')
  const [categoryId, setCategoryId] = useState<number | null>(null)
  const [digits, setDigits] = useState('')
  const [occurredOn, setOccurredOn] = useState(() => isoDaysAgo(0))
  const [description, setDescription] = useState('')
  const [amountError, setAmountError] = useState<string | null>(null)
  const [categoryError, setCategoryError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const amountRef = useRef<HTMLInputElement>(null)

  // After the frame, so a sheet's showModal() has run and won't move focus back to its first button.
  useEffect(() => {
    if (!autoFocusAmount) return
    const frame = requestAnimationFrame(() => amountRef.current?.focus())
    return () => cancelAnimationFrame(frame)
  }, [autoFocusAmount])

  const options = categories.filter((c) => c.type === type)
  const today = isoDaysAgo(0)
  const yesterday = isoDaysAgo(1)
  const expense = type === 'expense'

  function changeType(next: Transaction['type']) {
    setType(next)
    setCategoryId(null)
    // The usual order is type, then amount, so an empty amount takes focus straight away.
    if (!digits) amountRef.current?.focus()
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const amount = Number(digits)
    const missingAmount = !amount
    setAmountError(missingAmount ? 'Enter an amount.' : null)
    setCategoryError(categoryId ? null : 'Choose a category.')
    if (missingAmount || !categoryId) return
    try {
      await createTransaction.mutateAsync({ categoryId, amount, occurredOn, description, type })
      setDigits('')
      setDescription('')
      toast.success(expense ? 'Expense added' : 'Income added')
      onAdded?.()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the transaction.')
    }
  }

  return (
    <form onSubmit={onSubmit} noValidate className="space-y-5">
      <SegmentedControl
        label="Transaction type"
        value={type}
        onChange={changeType}
        size="tall"
        fullWidth
        options={[
          { value: 'expense', label: 'Expense', icon: <ArrowUpRight aria-hidden="true" />, selectedClass: 'bg-accent text-on-accent' },
          { value: 'income', label: 'Income', icon: <ArrowDownLeft aria-hidden="true" />, selectedClass: 'bg-income text-on-income' },
        ]}
      />

      <Field label="Amount" error={amountError}>
        {(control) => (
          <MoneyInput
            {...control}
            ref={amountRef}
            type={type}
            invalid={Boolean(amountError)}
            enterKeyHint="next"
            digits={digits}
            onDigitsChange={(next) => {
              setDigits(next)
              setAmountError(null)
            }}
          />
        )}
      </Field>

      <fieldset aria-describedby={categoryError ? `${groupName}-category-error` : undefined}>
        <legend className={labelClass}>Category</legend>
        <div className="flex flex-wrap gap-2">
          {options.map((c) => {
            const selected = c.id === categoryId
            return (
              <label key={c.id} className={`${chipClass(selected, type, 'pr-4 pl-3')} cursor-pointer has-focus-visible:outline-3 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent`}>
                <input
                  type="radio"
                  name={`${groupName}-category`}
                  value={c.id}
                  checked={selected}
                  onChange={() => {
                    setCategoryId(c.id)
                    setCategoryError(null)
                  }}
                  className="sr-only"
                />
                <span aria-hidden="true" className="size-3 shrink-0 rounded-full" style={{ backgroundColor: c.color }} />
                {c.name}
              </label>
            )
          })}
        </div>
        <FieldErrorText id={`${groupName}-category-error`}>{categoryError}</FieldErrorText>
      </fieldset>

      <fieldset>
        <legend className={labelClass}>Date</legend>
        <div className="flex flex-wrap gap-2">
          {[
            { label: 'Today', value: today },
            { label: 'Yesterday', value: yesterday },
          ].map((d) => (
            <button
              key={d.label}
              type="button"
              aria-pressed={occurredOn === d.value}
              onClick={() => setOccurredOn(d.value)}
              className={chipClass(occurredOn === d.value, type)}
            >
              {d.label}
            </button>
          ))}
          <label className={`${chipClass(occurredOn !== today && occurredOn !== yesterday, type, 'pr-2 pl-4')} relative min-w-[150px] flex-1 cursor-pointer has-focus-visible:outline-3 has-focus-visible:outline-offset-2 has-focus-visible:outline-accent`}>
            <CalendarDays aria-hidden="true" className="size-[18px] shrink-0" />
            <span className="sr-only">Pick a date</span>
            <input
              type="date"
              required
              value={occurredOn}
              onChange={(e) => e.target.value && setOccurredOn(e.target.value)}
              className="tabular min-w-0 flex-1 bg-transparent text-[14px] font-semibold focus-visible:outline-none"
            />
          </label>
        </div>
      </fieldset>

      <Field label="Note (optional)">
        {(control) => (
          <input
            {...control}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Cà phê với Minh"
            enterKeyHint="done"
            className={inputClass}
          />
        )}
      </Field>

      {error && (
        <div role="alert">
          <FieldErrorText>{error}</FieldErrorText>
        </div>
      )}

      <button
        type="submit"
        disabled={createTransaction.isPending}
        aria-busy={createTransaction.isPending}
        className={`${buttonClass(expense ? 'primary' : 'income', 'lg')} w-full`}
      >
        <Plus aria-hidden="true" />
        {createTransaction.isPending ? 'Adding…' : expense ? 'Add expense' : 'Add income'}
      </button>
    </form>
  )
}

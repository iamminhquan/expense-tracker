import { useState, type FormEvent } from 'react'
import { Check } from 'lucide-react'
import { useUpdateTransaction } from '../../hooks/useTransactions'
import { AmountInput } from '../../components/ui/AmountInput'
import { Field } from '../../components/ui/Field'
import { MoneyInput } from '../../components/ui/MoneyInput'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { SelectControl } from '../../components/ui/SelectControl'
import { ApiError } from '../../lib/api/client'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import type { Category, Transaction } from '../../lib/api/types'

interface EditTransactionFormProps {
  transaction: Transaction
  categories: Category[]
  /** Inline is the desktop row turned into fields; stacked is the phone sheet's form. */
  layout: 'inline' | 'stacked'
  onDone: () => void
}

export function EditTransactionForm({ transaction, categories, layout, onDone }: EditTransactionFormProps) {
  const updateTransaction = useUpdateTransaction()
  const toast = useToast()
  const [amount, setAmount] = useState(String(transaction.amount))
  const [categoryId, setCategoryId] = useState(transaction.categoryId)
  const [occurredOn, setOccurredOn] = useState(transaction.occurredOn)
  const [description, setDescription] = useState(transaction.description)
  const [error, setError] = useState<string | null>(null)

  const rowName = transaction.description || transaction.categoryName
  const sameTypeCategories = categories.filter((c) => c.type === transaction.type)

  async function onSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateTransaction.mutateAsync({ id: transaction.id, input: { categoryId, amount: Number(amount), occurredOn, description } })
      onDone()
      toast.success('Changes saved')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update the transaction.')
    }
  }

  const categorySelect = (props: object) => (
    <SelectControl {...props} value={categoryId} onChange={(e) => setCategoryId(Number(e.target.value))}>
      {sameTypeCategories.map((c) => (
        <option key={c.id} value={c.id}>
          {c.name}
        </option>
      ))}
    </SelectControl>
  )
  const size = layout === 'inline' ? 'md' : 'lg'
  const cancel = (
    <button type="button" onClick={onDone} className={buttonClass(layout === 'inline' ? 'ghost' : 'secondary', size)}>
      Cancel
    </button>
  )
  const save = (
    <button type="submit" disabled={updateTransaction.isPending} aria-busy={updateTransaction.isPending} className={buttonClass('primary', size)}>
      <Check aria-hidden="true" />
      Save
    </button>
  )
  const errorText = error && (
    <div role="alert" className={layout === 'inline' ? 'px-5 pb-3' : ''}>
      <FieldErrorText>{error}</FieldErrorText>
    </div>
  )

  if (layout === 'inline') {
    return (
      <>
        <form
          onSubmit={onSave}
          onKeyDown={(e) => {
            if (e.key === 'Escape') onDone()
          }}
          aria-label={`Edit ${rowName}`}
          className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_200px] items-center gap-2.5 px-5 py-4"
        >
          <input autoFocus aria-label="Note" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Note" className={`${inputClass} col-span-2`} />
          <AmountInput required aria-label="Amount" value={amount} onChange={(e) => setAmount(e.target.value)} />
          {categorySelect({ 'aria-label': 'Category' })}
          <input type="date" required aria-label="Date" value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={`${inputClass} tabular`} />
          <div className="flex justify-end gap-1">
            {cancel}
            {save}
          </div>
        </form>
        {errorText}
      </>
    )
  }

  return (
    <form onSubmit={onSave} aria-label={`Edit ${rowName}`} className="grid gap-4">
      <Field label="Amount">
        {(control) => <MoneyInput {...control} autoFocus required type={transaction.type} digits={amount} onDigitsChange={setAmount} />}
      </Field>
      <Field label="Note">{(control) => <input {...control} value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />}</Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Category">{(control) => categorySelect(control)}</Field>
        <Field label="Date">
          {(control) => <input {...control} type="date" required value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} className={`${inputClass} tabular`} />}
        </Field>
      </div>
      {errorText}
      <div className="grid grid-cols-2 gap-3 pt-1">
        {cancel}
        {save}
      </div>
    </form>
  )
}

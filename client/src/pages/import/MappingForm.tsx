import { useState } from 'react'
import { ArrowRight } from 'lucide-react'
import { Banner } from '../../components/ui/Banner'
import { Checkbox } from '../../components/ui/Checkbox'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { SelectControl } from '../../components/ui/SelectControl'
import type { MappingFields, MappingNeeded } from '../../lib/api/import'
import { buttonClass, inputClass } from '../../lib/formStyles'

const emptyMapping = (guess?: MappingFields): MappingFields =>
  guess ?? {
    dateCol: 0, amountCol: 1, typeCol: -1, categoryCol: -1, noteCol: -1,
    dateLayout: '2006-01-02', negativeIsExpense: false, fallbackCategory: '',
  }

const ROLES: { label: string; key: 'dateCol' | 'amountCol' | 'typeCol' | 'categoryCol' | 'noteCol'; optional: boolean }[] = [
  { label: 'Date', key: 'dateCol', optional: false },
  { label: 'Amount', key: 'amountCol', optional: false },
  { label: 'Type', key: 'typeCol', optional: true },
  { label: 'Category', key: 'categoryCol', optional: true },
  { label: 'Note', key: 'noteCol', optional: true },
]

interface MappingFormProps {
  data: MappingNeeded
  submitting: boolean
  error: string | null
  onCancel: () => void
  onSubmit: (mapping: MappingFields) => void
}

export function MappingForm({ data, submitting, error, onCancel, onSubmit }: MappingFormProps) {
  const [mapping, setMapping] = useState<MappingFields>(emptyMapping(data.guess))

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        onSubmit(mapping)
      }}
      className="space-y-6"
    >
      <div>
        <h2 className="heading text-[20px] leading-[26px] text-ink">Which column holds what?</h2>
        <p className="mt-1 text-[14px] leading-[22px] text-ink-muted">We didn't recognize this file's layout. Here are its first rows.</p>
      </div>

      {data.ambiguousDate && (
        <Banner kind="warning" title="Check the date order">
          These dates could be read day-first or month-first. Pick the right format below.
        </Banner>
      )}

      <div tabIndex={0} role="region" aria-label="First rows of the file" className="overflow-x-auto rounded-panel border border-border">
        <table className="w-full text-[13px] leading-[18px]">
          <thead>
            <tr className="bg-surface-2">
              {data.columns.map((c) => (
                <th key={c} scope="col" className="px-3 py-2.5 text-left font-semibold whitespace-nowrap text-ink-muted">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.sample.map((row, i) => (
              <tr key={i} className="border-t border-border">
                {row.map((cell, j) => (
                  <td key={j} className="tabular px-3 py-2.5 whitespace-nowrap text-ink">
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {ROLES.map((role) => (
          <Field key={role.key} label={role.optional ? `${role.label} (optional)` : role.label}>
            {(control) => (
              <SelectControl
                {...control}
                required={!role.optional}
                value={mapping[role.key]}
                onChange={(e) => setMapping((m) => ({ ...m, [role.key]: Number(e.target.value) }))}
              >
                <option value={-1}>{role.optional ? 'None' : 'Choose a column'}</option>
                {data.columns.map((c, i) => (
                  <option key={i} value={i}>
                    {c}
                  </option>
                ))}
              </SelectControl>
            )}
          </Field>
        ))}
        <Field label="Date format">
          {(control) => (
            <SelectControl {...control} value={mapping.dateLayout} onChange={(e) => setMapping((m) => ({ ...m, dateLayout: e.target.value }))}>
              {data.dateFormats.map((f) => (
                <option key={f.key} value={f.key}>
                  {f.label}
                </option>
              ))}
            </SelectControl>
          )}
        </Field>
      </div>

      <Checkbox checked={mapping.negativeIsExpense} onChange={(checked) => setMapping((m) => ({ ...m, negativeIsExpense: checked }))}>
        A negative amount means an expense
      </Checkbox>

      {mapping.categoryCol === -1 && (
        <Field label="File every row under" hint="Pick an existing category, or type a new name to create it.">
          {(control) => (
            <>
              <input
                {...control}
                list="category-names"
                value={mapping.fallbackCategory}
                onChange={(e) => setMapping((m) => ({ ...m, fallbackCategory: e.target.value }))}
                placeholder="Category name"
                className={inputClass}
              />
              <datalist id="category-names">
                {data.categoryNames.map((n) => (
                  <option key={n} value={n} />
                ))}
              </datalist>
            </>
          )}
        </Field>
      )}

      {error && (
        <div role="alert">
          <FieldErrorText>{error}</FieldErrorText>
        </div>
      )}

      <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <button type="button" onClick={onCancel} className={buttonClass('ghost')}>
          Start over
        </button>
        <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary')} max-sm:h-[52px]`}>
          {submitting ? 'Checking…' : 'Preview import'}
          <ArrowRight aria-hidden="true" />
        </button>
      </div>
    </form>
  )
}

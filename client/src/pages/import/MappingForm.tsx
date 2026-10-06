import { useState } from 'react'
import type { MappingFields, MappingNeeded } from '../../lib/api/import'
import { primaryButtonClass } from '../../lib/formStyles'

const emptyMapping = (guess?: MappingFields): MappingFields =>
  guess ?? {
    dateCol: 0, amountCol: 1, typeCol: -1, categoryCol: -1, noteCol: -1,
    dateLayout: '2006-01-02', negativeIsExpense: false, fallbackCategory: '',
  }

interface MappingFormProps {
  data: MappingNeeded
  submitting: boolean
  error: string | null
  onCancel: () => void
  onSubmit: (mapping: MappingFields) => void
}

export function MappingForm({ data, submitting, error, onCancel, onSubmit }: MappingFormProps) {
  const [mapping, setMapping] = useState<MappingFields>(emptyMapping(data.guess))

  const roles: { label: string; key: keyof MappingFields; optional: boolean }[] = [
    { label: 'Date', key: 'dateCol', optional: false },
    { label: 'Amount', key: 'amountCol', optional: false },
    { label: 'Type', key: 'typeCol', optional: true },
    { label: 'Category', key: 'categoryCol', optional: true },
    { label: 'Note', key: 'noteCol', optional: true },
  ]

  return (
    <div className="rounded-[16px] border border-border-card bg-surface p-6">
      <p className="mb-1 text-[14px] font-semibold">We don't recognize this format</p>
      <p className="mb-4 text-[13px] text-ink-muted">Tell us which column holds what. {data.rows} rows found.</p>

      {data.ambiguousDate && (
        <p className="mb-4 rounded-[10px] bg-track p-3 text-[13px] text-ink-muted">
          The dates in this file could be read either day-first or month-first -- double check the order below.
        </p>
      )}

      <div className="mb-4 overflow-x-auto rounded-[10px] border border-border-list">
        <table className="w-full text-[12px]">
          <thead>
            <tr className="border-b border-border-list bg-track">
              {data.columns.map((c) => (
                <th key={c} className="px-2 py-1 text-left font-medium text-ink-faint">
                  {c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.sample.map((row, i) => (
              <tr key={i} className="border-b border-border-list last:border-0">
                {row.map((cell, j) => (
                  <td key={j} className="px-2 py-1 text-ink-muted">
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="space-y-3">
        {roles.map((role) => (
          <div key={role.key} className="flex items-center gap-3">
            <label className="w-[70px] text-[13px] text-ink-muted">
              {role.label}
              {role.optional ? '' : ' *'}
            </label>
            <select
              value={mapping[role.key] as number}
              onChange={(e) => setMapping((m) => ({ ...m, [role.key]: Number(e.target.value) }))}
              className="flex-1 rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
            >
              <option value={-1}>{role.optional ? 'None' : 'Choose a column'}</option>
              {data.columns.map((c, i) => (
                <option key={i} value={i}>
                  {c}
                </option>
              ))}
            </select>
          </div>
        ))}

        <div className="flex items-center gap-3">
          <label className="w-[70px] text-[13px] text-ink-muted">Dates are</label>
          <select
            value={mapping.dateLayout}
            onChange={(e) => setMapping((m) => ({ ...m, dateLayout: e.target.value }))}
            className="flex-1 rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
          >
            {data.dateFormats.map((f) => (
              <option key={f.key} value={f.key}>
                {f.label}
              </option>
            ))}
          </select>
        </div>

        <label className="flex items-center gap-2 text-[13px] text-ink-muted">
          <input
            type="checkbox"
            checked={mapping.negativeIsExpense}
            onChange={(e) => setMapping((m) => ({ ...m, negativeIsExpense: e.target.checked }))}
          />
          A negative amount means an expense
        </label>

        {mapping.categoryCol === -1 && (
          <div className="flex items-center gap-3">
            <label className="w-[70px] text-[13px] text-ink-muted">File under</label>
            <input
              list="category-names"
              value={mapping.fallbackCategory}
              onChange={(e) => setMapping((m) => ({ ...m, fallbackCategory: e.target.value }))}
              placeholder="Category name"
              className="flex-1 rounded-[8px] border border-border-input bg-surface px-2 py-1 text-[13px]"
            />
            <datalist id="category-names">
              {data.categoryNames.map((n) => (
                <option key={n} value={n} />
              ))}
            </datalist>
          </div>
        )}
      </div>

      {error && <p className="mt-3 text-[13px] text-expense">{error}</p>}

      <div className="mt-5 flex gap-2">
        <button onClick={onCancel} className="rounded-[10px] px-4 py-2 text-[13px] text-ink-faint hover:bg-track">
          Start over
        </button>
        <button onClick={() => onSubmit(mapping)} disabled={submitting} className={primaryButtonClass}>
          {submitting ? 'Checking…' : 'Preview import'}
        </button>
      </div>
    </div>
  )
}

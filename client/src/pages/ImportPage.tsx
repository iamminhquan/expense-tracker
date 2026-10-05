import { useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  importTransactions,
  type ImportPreview,
  type ImportResult,
  type MappingFields,
  type MappingNeeded,
} from '../lib/api/import'
import { ApiError } from '../lib/api/client'
import { primaryButtonClass } from '../components/layout/AuthLayout'

type Step =
  | { kind: 'upload' }
  | { kind: 'mapping'; file: File; data: MappingNeeded }
  | { kind: 'preview'; file: File; data: ImportPreview; mapping?: MappingFields }
  | { kind: 'done'; data: ImportResult }

const emptyMapping = (guess?: MappingFields): MappingFields =>
  guess ?? {
    dateCol: 0, amountCol: 1, typeCol: -1, categoryCol: -1, noteCol: -1,
    dateLayout: '2006-01-02', negativeIsExpense: false, fallbackCategory: '',
  }

export function ImportPage() {
  const [step, setStep] = useState<Step>({ kind: 'upload' })
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  async function handleFile(file: File) {
    setError(null)
    setSubmitting(true)
    try {
      const result = await importTransactions(file)
      if ('needsMapping' in result) {
        setStep({ kind: 'mapping', file, data: result })
      } else if ('preview' in result) {
        setStep({ kind: 'preview', file, data: result })
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not read that file.')
    } finally {
      setSubmitting(false)
    }
  }

  async function submitMapping(file: File, mapping: MappingFields) {
    setError(null)
    setSubmitting(true)
    try {
      const result = await importTransactions(file, { mapping })
      if ('preview' in result) {
        setStep({ kind: 'preview', file, data: result, mapping })
      } else if ('needsMapping' in result) {
        setStep({ kind: 'mapping', file, data: result })
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not read that file.')
    } finally {
      setSubmitting(false)
    }
  }

  async function confirmImport(file: File, mapping: MappingFields | undefined, fingerprint: string) {
    setError(null)
    setSubmitting(true)
    try {
      const result = await importTransactions(file, { mapping, confirm: true, fingerprint })
      if ('imported' in result) {
        setStep({ kind: 'done', data: result })
        void queryClient.invalidateQueries({ queryKey: ['transactions'] })
        void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
        void queryClient.invalidateQueries({ queryKey: ['categories'] })
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not import that file.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-[640px] space-y-6">
      <h1 className="text-[20px] font-semibold">Import transactions</h1>

      {step.kind === 'upload' && (
        <div className="rounded-[16px] border border-border-card bg-surface p-6">
          <p className="mb-4 text-[13px] text-ink-muted">
            Upload a .csv file -- either one exported from $pend, or one from your bank or another app. We'll ask how to
            read it if we can't tell automatically.
          </p>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) void handleFile(file)
            }}
            className="text-[13px]"
          />
          {submitting && <p className="mt-3 text-[13px] text-ink-faint">Reading file…</p>}
          {error && <p className="mt-3 text-[13px] text-expense">{error}</p>}
        </div>
      )}

      {step.kind === 'mapping' && (
        <MappingForm
          data={step.data}
          submitting={submitting}
          error={error}
          onCancel={() => setStep({ kind: 'upload' })}
          onSubmit={(mapping) => void submitMapping(step.file, mapping)}
        />
      )}

      {step.kind === 'preview' && (
        <PreviewPanel
          data={step.data}
          submitting={submitting}
          error={error}
          onBack={() => setStep({ kind: 'upload' })}
          onConfirm={() => void confirmImport(step.file, step.mapping, step.data.fingerprint)}
        />
      )}

      {step.kind === 'done' && (
        <div className="rounded-[16px] border border-border-card bg-surface p-6 text-center">
          <p className="mb-4 text-[14px] text-income">
            Imported {step.data.imported} transaction{step.data.imported === 1 ? '' : 's'}.
          </p>
          <button onClick={() => navigate(`/transactions?month=${step.data.month}`)} className={primaryButtonClass}>
            View transactions
          </button>
        </div>
      )}
    </div>
  )
}

function MappingForm({
  data,
  submitting,
  error,
  onCancel,
  onSubmit,
}: {
  data: MappingNeeded
  submitting: boolean
  error: string | null
  onCancel: () => void
  onSubmit: (mapping: MappingFields) => void
}) {
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

function PreviewPanel({
  data,
  submitting,
  error,
  onBack,
  onConfirm,
}: {
  data: ImportPreview
  submitting: boolean
  error: string | null
  onBack: () => void
  onConfirm: () => void
}) {
  return (
    <div className="rounded-[16px] border border-border-card bg-surface p-6">
      <p className="mb-1 text-[14px] font-semibold">Preview</p>
      <p className="mb-4 text-[13px] text-ink-muted">
        {data.rowCount} row{data.rowCount === 1 ? '' : 's'} ready to import.
        {data.rounded > 0 && ` ${data.rounded} amount(s) rounded to the nearest đồng.`}
        {data.duplicates > 0 && ` ${data.duplicates} look like duplicates of transactions already on file.`}
      </p>

      {data.newCategories.length > 0 && (
        <p className="mb-3 text-[13px] text-ink-muted">
          New categories to create: {data.newCategories.map((c) => c.name).join(', ')}
        </p>
      )}

      {data.errors.length > 0 && (
        <div className="mb-4 rounded-[10px] bg-danger-tint p-3">
          <p className="mb-1 text-[13px] font-semibold text-expense">
            {data.dateSuspect ? 'Most failures look like a date-order mistake above.' : `${data.errors.length} line(s) can't be imported:`}
          </p>
          <ul className="max-h-[160px] space-y-0.5 overflow-y-auto text-[12px] text-ink-muted">
            {data.errors.map((e, i) => (
              <li key={i}>
                Line {e.line}: {e.message}
              </li>
            ))}
          </ul>
          {data.moreErrors > 0 && <p className="mt-1 text-[12px] text-ink-faint">...and {data.moreErrors} more.</p>}
        </div>
      )}

      {error && <p className="mb-3 text-[13px] text-expense">{error}</p>}

      <div className="flex gap-2">
        <button onClick={onBack} className="rounded-[10px] px-4 py-2 text-[13px] text-ink-faint hover:bg-track">
          Start over
        </button>
        <button onClick={onConfirm} disabled={submitting || !data.importable} className={primaryButtonClass}>
          {submitting ? 'Importing…' : `Import ${data.rowCount} row${data.rowCount === 1 ? '' : 's'}`}
        </button>
      </div>
    </div>
  )
}

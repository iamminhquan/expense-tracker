import { useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  importTransactions,
  type ImportPreview,
  type ImportResult,
  type MappingFields,
  type MappingNeeded,
} from '../../lib/api/import'
import { ApiError } from '../../lib/api/client'
import { primaryButtonClass } from '../../lib/formStyles'
import { MappingForm } from './MappingForm'
import { PreviewPanel } from './PreviewPanel'

type Step =
  | { kind: 'upload' }
  | { kind: 'mapping'; file: File; data: MappingNeeded }
  | { kind: 'preview'; file: File; data: ImportPreview; mapping?: MappingFields }
  | { kind: 'done'; data: ImportResult }

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

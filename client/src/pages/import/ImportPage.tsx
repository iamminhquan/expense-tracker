import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ChevronLeft, CircleCheck } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import {
  importTransactions,
  type ImportPreview,
  type ImportResult,
  type MappingFields,
  type MappingNeeded,
} from '../../lib/api/import'
import { ApiError } from '../../lib/api/client'
import { buttonClass, cardClass, pageTitleClass } from '../../lib/formStyles'
import { FileRow } from './FileRow'
import { ImportStepper } from './ImportStepper'
import { MappingForm } from './MappingForm'
import { PreviewPanel } from './PreviewPanel'
import { UploadStep } from './UploadStep'

type Step =
  | { kind: 'upload' }
  | { kind: 'mapping'; file: File; data: MappingNeeded }
  | { kind: 'preview'; file: File; data: ImportPreview; mapping?: MappingFields }
  | { kind: 'done'; data: ImportResult }

export function ImportPage() {
  const [step, setStep] = useState<Step>({ kind: 'upload' })
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  function startOver() {
    setError(null)
    setStep({ kind: 'upload' })
  }

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

  const stepIndex = { upload: 0, mapping: 1, preview: 2, done: 3 }[step.kind]

  return (
    <div className="mx-auto max-w-[800px] space-y-4 md:space-y-6">
      <div>
        <Link to="/transactions" className="mb-2 inline-flex items-center gap-1 text-[14px] font-semibold text-ink-muted hover:text-ink">
          <ChevronLeft aria-hidden="true" className="size-4" />
          Transactions
        </Link>
        <h1 className={pageTitleClass}>Import transactions</h1>
      </div>

      <div className={`${cardClass} space-y-6`}>
        <ImportStepper current={stepIndex} />

        {step.kind === 'upload' && <UploadStep reading={submitting} error={error} onFile={(file) => void handleFile(file)} />}

        {(step.kind === 'mapping' || step.kind === 'preview') && (
          <FileRow file={step.file} rows={step.kind === 'mapping' ? step.data.rows : undefined} onRemove={startOver} />
        )}

        {step.kind === 'mapping' && (
          <MappingForm
            data={step.data}
            submitting={submitting}
            error={error}
            onCancel={startOver}
            onSubmit={(mapping) => void submitMapping(step.file, mapping)}
          />
        )}

        {step.kind === 'preview' && (
          <PreviewPanel
            data={step.data}
            submitting={submitting}
            error={error}
            onBack={startOver}
            onConfirm={() => void confirmImport(step.file, step.mapping, step.data.fingerprint)}
          />
        )}

        {step.kind === 'done' && (
          <div className="flex flex-col items-center py-6 text-center">
            <span aria-hidden="true" className="mb-4 flex size-[72px] items-center justify-center rounded-full bg-income-tint text-income">
              <CircleCheck className="size-9" />
            </span>
            <p role="status" className="font-display text-[22px] leading-7 font-bold text-ink">
              Imported {step.data.imported} transaction{step.data.imported === 1 ? '' : 's'}
            </p>
            <p className="mt-1.5 text-[14px] leading-[22px] text-ink-muted">Rows that matched earlier transactions are marked "Possible duplicate".</p>
            <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row">
              <button type="button" onClick={startOver} className={buttonClass('secondary')}>
                Import another file
              </button>
              <button type="button" onClick={() => navigate(`/transactions?month=${step.data.month}`)} className={buttonClass('primary')}>
                View transactions
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

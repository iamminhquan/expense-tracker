import type { ImportPreview } from '../../lib/api/import'
import { primaryButtonClass } from '../../lib/formStyles'

interface PreviewPanelProps {
  data: ImportPreview
  submitting: boolean
  error: string | null
  onBack: () => void
  onConfirm: () => void
}

export function PreviewPanel({ data, submitting, error, onBack, onConfirm }: PreviewPanelProps) {
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

import { CircleAlert } from 'lucide-react'
import { Badge } from '../../components/ui/Badge'
import { Banner } from '../../components/ui/Banner'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import type { ImportPreview } from '../../lib/api/import'
import { buttonClass } from '../../lib/formStyles'

interface PreviewPanelProps {
  data: ImportPreview
  submitting: boolean
  error: string | null
  onBack: () => void
  onConfirm: () => void
}

function plural(n: number, word: string): string {
  return `${n} ${word}${n === 1 ? '' : 's'}`
}

function StatTile({ label, value, danger }: { label: string; value: number; danger?: boolean }) {
  return (
    <div className={`rounded-panel px-4 py-3.5 ${danger ? 'bg-danger-tint' : 'bg-surface-2'}`}>
      <p className="text-[13px] leading-[18px] text-ink-muted">{label}</p>
      <p className={`figure mt-1 text-[34px] leading-none ${danger ? 'text-danger' : 'text-ink'}`}>{value}</p>
    </div>
  )
}

export function PreviewPanel({ data, submitting, error, onBack, onConfirm }: PreviewPanelProps) {
  const errorCount = data.errors.length + data.moreErrors

  return (
    <div className="space-y-6">
      <div>
        <h2 className="heading text-[20px] leading-[26px] text-ink">Check before importing</h2>
        <p className="mt-1 text-[14px] leading-[22px] text-ink-muted">
          Nothing is saved until you confirm. An import is all or nothing: a single bad line stops the whole file.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatTile label="Rows ready" value={data.rowCount} />
        <StatTile label="Amounts rounded" value={data.rounded} />
        <StatTile label="Possible duplicates" value={data.duplicates} />
        <StatTile label="Lines with errors" value={errorCount} danger={errorCount > 0} />
      </div>

      {data.newCategories.length > 0 && (
        <div>
          <p className="mb-2 text-[14px] leading-5 font-bold text-ink">New categories to create</p>
          <div className="flex flex-wrap gap-2">
            {data.newCategories.map((c) => (
              <Badge key={`${c.type}:${c.name}`} kind={c.type === 'income' ? 'income' : 'expense'}>
                {c.name} · {c.type}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {data.duplicates > 0 && (
        <Banner kind="warning" title={`${plural(data.duplicates, 'row')} ${data.duplicates === 1 ? 'matches a transaction' : 'match transactions'} already on file`}>
          They'll still be imported, and marked "Possible duplicate" in your list so you can check them.
        </Banner>
      )}

      {data.rounded > 0 && <Banner kind="info">{plural(data.rounded, 'amount')} will be rounded to the nearest đồng.</Banner>}

      {data.errors.length > 0 && (
        <div className="overflow-hidden rounded-panel border border-danger/50">
          <p className="flex items-center gap-2 bg-danger-tint px-4 py-2.5 text-[14px] leading-5 font-bold text-ink">
            <CircleAlert aria-hidden="true" className="size-[18px] text-danger" />
            {data.dateSuspect ? 'Most failures look like the wrong date order.' : `${plural(errorCount, 'line')} can't be imported`}
          </p>
          <ul tabIndex={0} aria-label="Lines that can't be imported" className="max-h-[198px] overflow-y-auto">
            {data.errors.map((e, i) => (
              <li key={i} className="flex min-h-11 items-center gap-3 border-t border-border px-4 py-2 text-[14px] leading-5 text-ink">
                <span className="tabular w-16 shrink-0 font-semibold text-ink-muted">Line {e.line}</span>
                {e.message}
              </li>
            ))}
            {data.moreErrors > 0 && <li className="border-t border-border px-4 py-2.5 text-[13px] text-ink-muted">…and {data.moreErrors} more.</li>}
          </ul>
        </div>
      )}

      {!data.importable && (
        <Banner kind="danger" title="Nothing can be imported yet">
          Fix these lines in the file and upload it again, or go back and change how its columns are read.
        </Banner>
      )}

      {error && (
        <div role="alert">
          <FieldErrorText>{error}</FieldErrorText>
        </div>
      )}

      <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <button type="button" onClick={onBack} className={buttonClass('ghost')}>
          Start over
        </button>
        <button type="button" onClick={onConfirm} disabled={submitting || !data.importable} aria-busy={submitting} className={`${buttonClass('primary')} max-sm:h-[52px]`}>
          {submitting ? 'Importing…' : `Import ${plural(data.rowCount, 'row')}`}
        </button>
      </div>
    </div>
  )
}

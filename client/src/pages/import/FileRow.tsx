import { FileText, X } from 'lucide-react'
import { iconButtonClass } from '../../lib/formStyles'

function formatSize(bytes: number): string {
  return bytes < 1024 ? `${bytes} B` : `${Math.round(bytes / 1024)} KB`
}

export function FileRow({ file, rows, onRemove }: { file: File; rows?: number; onRemove: () => void }) {
  return (
    <div className="flex items-center gap-3 rounded-card bg-surface-2 p-3 pr-1.5">
      <span aria-hidden="true" className="flex size-12 shrink-0 items-center justify-center rounded-control bg-surface text-accent-text">
        <FileText className="size-6" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{file.name}</p>
        <p className="tabular text-[13px] leading-[18px] text-ink-muted">
          {formatSize(file.size)}
          {rows !== undefined && ` · ${rows} rows`}
        </p>
      </div>
      <button type="button" onClick={onRemove} aria-label={`Remove ${file.name} and start over`} className={iconButtonClass}>
        <X aria-hidden="true" />
      </button>
    </div>
  )
}

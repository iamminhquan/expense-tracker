import { CloudOff, RotateCw } from 'lucide-react'
import { buttonClass } from '../../lib/formStyles'

export function InlineError({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="mx-auto mt-6 flex max-w-md animate-fade-in flex-col items-center rounded-card border border-border bg-surface px-6 py-10 text-center shadow-card">
      <span aria-hidden="true" className="mb-4 flex size-14 items-center justify-center rounded-[18px] bg-danger-tint text-danger">
        <CloudOff className="size-7" />
      </span>
      <p className="font-display text-[20px] leading-[26px] font-semibold tracking-[-0.01em] text-ink">{message}</p>
      <p className="mt-1.5 text-[14px] leading-[22px] text-ink-muted">Check your connection, then try again.</p>
      {onRetry && (
        <button type="button" onClick={onRetry} className={`${buttonClass('primary')} mt-6`}>
          <RotateCw aria-hidden="true" />
          Retry
        </button>
      )}
    </div>
  )
}

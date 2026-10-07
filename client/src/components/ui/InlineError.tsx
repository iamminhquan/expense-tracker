import { CloudOff, RotateCw } from 'lucide-react'
import { buttonClass } from '../../lib/formStyles'

export function InlineError({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="flex animate-fade-in flex-col items-center rounded-tile border border-border bg-surface px-6 py-10 text-center">
      <span aria-hidden="true" className="mb-4 flex size-14 items-center justify-center rounded-full bg-danger-tint text-danger">
        <CloudOff className="size-6" />
      </span>
      <p className="heading text-[19px] leading-[26px] text-ink">{message}</p>
      <p className="mt-1.5 max-w-[340px] text-[14px] leading-[21px] text-ink-muted">Check your connection, then try again. Nothing you saved is lost.</p>
      {onRetry && (
        <button type="button" onClick={onRetry} className={`${buttonClass('primary')} mt-6`}>
          <RotateCw aria-hidden="true" />
          Retry
        </button>
      )}
    </div>
  )
}

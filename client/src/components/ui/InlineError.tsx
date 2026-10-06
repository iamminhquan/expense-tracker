import { CircleAlert, RotateCw } from 'lucide-react'
import { buttonClass } from '../../lib/formStyles'

export function InlineError({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="flex items-center gap-3 rounded-[16px] border border-danger bg-danger-tint px-4 py-3">
      <CircleAlert aria-hidden="true" className="size-5 shrink-0 text-danger" />
      <p className="flex-1 text-[14px] leading-5 font-semibold text-ink">{message}</p>
      {onRetry && (
        <button type="button" onClick={onRetry} className={buttonClass('secondary', 'sm')}>
          <RotateCw aria-hidden="true" />
          Retry
        </button>
      )}
    </div>
  )
}

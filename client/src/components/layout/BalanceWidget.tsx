import type { Balance } from '../../lib/api/types'
import { formatVND } from '../../lib/format'

// spentPct alone can't tell "no income" from "nothing spent yet", hence hasIncome.
function ratioLabel(balance: Balance): string {
  if (!balance.hasIncome) return 'No income this month'
  if (balance.spentPct >= 100) return "Over this month's income"
  return `Spent ${balance.spentPct}% of this month's income`
}

const RADIUS = 15.5
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

export function BalanceWidget({ balance }: { balance: Balance }) {
  const amount = balance.empty ? formatVND(0) : `${balance.remaining < 0 ? '-' : ''}${formatVND(balance.remaining)}`
  const pct = balance.hasIncome ? Math.min(100, Math.max(0, balance.spentPct)) : 0
  const label = ratioLabel(balance)

  return (
    <div className="flex items-center gap-3">
      <svg viewBox="0 0 36 36" className="size-9 shrink-0 -rotate-90" role="img" aria-label={label}>
        <circle cx="18" cy="18" r={RADIUS} fill="none" strokeWidth="5" className="stroke-surface-2" />
        {pct > 0 && (
          <circle
            cx="18"
            cy="18"
            r={RADIUS}
            fill="none"
            strokeWidth="5"
            strokeLinecap="round"
            strokeDasharray={`${(pct / 100) * CIRCUMFERENCE} ${CIRCUMFERENCE}`}
            className="stroke-accent"
          />
        )}
      </svg>
      <div className="min-w-0">
        <p className={`font-display tabular text-[17px] leading-[22px] font-bold ${balance.remaining < 0 ? 'text-danger' : 'text-ink'}`}>
          <span className="sr-only">Left this month: </span>
          {amount}
        </p>
        <p aria-hidden="true" className="text-[12px] leading-4 text-ink-muted">
          {label}
        </p>
      </div>
    </div>
  )
}

import type { Balance } from '../../lib/api/types'
import { formatVND } from '../../lib/format'

// ratioLabel mirrors handlers.balanceSummary's dropped RatioLabel field --
// computed client-side now instead of shipped as a pre-formatted English
// sentence; see dashboard_handlers.go's balanceDTO comment for why
// hasIncome exists (spentPct alone can't tell "no income" apart from
// "income, nothing spent of it yet").
function ratioLabel(balance: Balance): string {
  if (!balance.hasIncome) return 'No income this month'
  if (balance.spentPct >= 100) return "Over this month's income"
  return `Spent ${balance.spentPct}% of this month's income`
}

/** The running-balance popover shared by both nav bars. Mirrors header_balance.html. */
export function BalanceWidget({ balance }: { balance: Balance }) {
  const amountClass = balance.empty ? 'text-ink-zero' : balance.remaining < 0 ? 'text-expense' : 'text-ink-muted'
  const amount = balance.empty ? '0₫' : formatVND(balance.remaining)

  return (
    <details className="relative shrink-0">
      <summary
        className={`cursor-pointer list-none rounded-[7px] px-2 py-1 font-mono text-[13px] font-medium hover:bg-track [&::-webkit-details-marker]:hidden ${amountClass}`}
      >
        {amount}
      </summary>
      <div className="absolute right-0 top-full z-50 mt-1 w-[200px] rounded-[14px] border border-border-card bg-surface p-3 shadow-lg">
        <p className="mb-2 text-[10px] font-semibold tracking-[0.08em] text-ink-faintest">LEFT THIS MONTH</p>
        <p
          className={`mb-2 font-mono text-[19px] font-semibold ${balance.empty ? 'text-ink-zero' : balance.remaining < 0 ? 'text-expense' : 'text-ink'}`}
          style={{ letterSpacing: '-0.02em' }}
        >
          {amount}
        </p>
        <div className="mb-2 h-[6px] overflow-hidden rounded-full bg-track">
          <div className="h-full rounded-full bg-expense" style={{ width: `${balance.spentPct}%` }} />
        </div>
        <p className="text-[12px] text-ink-faint">{ratioLabel(balance)}</p>
      </div>
    </details>
  )
}

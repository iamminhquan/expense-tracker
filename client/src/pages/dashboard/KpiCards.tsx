import type { ReactNode } from 'react'
import { ArrowDownLeft, Minus, Scale, TrendingDown, TrendingUp } from 'lucide-react'
import type { DashboardResponse, PieLegendEntry } from '../../lib/api/types'
import { formatVND } from '../../lib/format'

interface ComparisonProps {
  current: number
  previous: number
  hasPrevData: boolean
  /** For spending, going down is the good direction; for income it's going up. */
  lowerIsBetter?: boolean
}

function Comparison({ current, previous, hasPrevData, lowerIsBetter }: ComparisonProps) {
  if (!hasPrevData) return <p className="text-[13px] leading-[18px] text-ink-muted">No data for last month</p>
  const diff = current - previous
  const pct = previous === 0 ? null : Math.round((Math.abs(diff) / previous) * 100)
  const Icon = diff > 0 ? TrendingUp : diff < 0 ? TrendingDown : Minus
  const chip = diff === 0 ? 'unchanged' : pct === null ? (diff > 0 ? 'up' : 'down') : `${diff > 0 ? 'up' : 'down'} ${pct}%`
  const better = diff !== 0 && (lowerIsBetter ? diff < 0 : diff > 0)
  const tone = diff === 0 ? 'bg-surface-2 text-ink' : better ? 'bg-income-tint text-income' : 'bg-expense-tint text-expense'
  return (
    <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] leading-[18px] text-ink-muted">
      <span className={`inline-flex h-6 items-center gap-1 rounded-full px-2 text-[12px] font-semibold ${tone}`}>
        <Icon aria-hidden="true" className="size-3.5" />
        {chip}
      </span>
      <span>
        from <span className="tabular font-semibold text-ink">{formatVND(previous)}</span> last month
      </span>
    </p>
  )
}

function CategoryStrip({ legend }: { legend: PieLegendEntry[] }) {
  if (legend.length === 0) {
    return <div aria-hidden="true" className="h-3 rounded-full bg-surface-2" />
  }
  const top = legend.slice(0, 3)
  return (
    <div>
      <div aria-hidden="true" className="flex h-3 gap-0.5 overflow-hidden rounded-full">
        {legend.map((entry) => (
          <span key={entry.name} className="h-full min-w-1 first:rounded-l-full last:rounded-r-full" style={{ flexGrow: entry.amount, backgroundColor: entry.color }} />
        ))}
      </div>
      <ul aria-label="Biggest categories" className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5">
        {top.map((entry) => (
          <li key={entry.name} className="flex min-w-0 items-center gap-1.5 text-[13px] leading-[18px]">
            <span aria-hidden="true" className="size-2.5 shrink-0 rounded-full" style={{ backgroundColor: entry.color }} />
            <span className="max-w-[140px] truncate font-semibold text-ink">{entry.name}</span>
            <span className="tabular text-ink-muted">{entry.percent}%</span>
          </li>
        ))}
        {legend.length > top.length && <li className="text-[13px] leading-[18px] text-ink-muted">+{legend.length - top.length} more</li>}
      </ul>
    </div>
  )
}

interface StatTileProps {
  label: string
  icon: ReactNode
  value: string
  inverted?: boolean
  valueClass?: string
  children: ReactNode
}

function StatTile({ label, icon, value, inverted, valueClass = '', children }: StatTileProps) {
  return (
    <div
      className={`flex min-w-0 flex-col justify-between gap-5 rounded-tile p-4 sm:p-6 ${inverted ? 'bg-ink text-app' : 'border border-border bg-surface text-ink'}`}
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-[14px] leading-5 font-semibold">{label}</p>
        <span aria-hidden="true" className={`flex size-8 items-center justify-center rounded-full [&_svg]:size-4 ${inverted ? 'bg-app/12' : 'bg-income-tint text-income'}`}>
          {icon}
        </span>
      </div>
      <div className="min-w-0">
        <p className={`figure text-[clamp(22px,6.4vw,44px)] md:text-[clamp(30px,3.2vw,46px)] leading-none [overflow-wrap:anywhere] ${valueClass}`}>{value}</p>
        <div className={`mt-2 text-[13px] leading-[18px] ${inverted ? 'opacity-75' : 'text-ink-muted'}`}>{children}</div>
      </div>
    </div>
  )
}

export function KpiCards({ data }: { data: DashboardResponse }) {
  const net = data.totalIncome - data.totalExpense
  return (
    <section aria-label="Month totals" className="grid gap-3 md:grid-cols-12 md:gap-5">
      <div className="flex min-w-0 flex-col gap-6 rounded-tile border border-border bg-surface p-5 sm:p-7 md:col-span-7 md:justify-between">
        <div>
          <p className="text-[15px] leading-[22px] font-semibold text-ink">Spent in {data.monthLabel}</p>
          <p className="figure mt-2 text-[clamp(52px,17vw,96px)] leading-[0.95] md:text-[clamp(64px,7.4vw,112px)] text-ink [overflow-wrap:anywhere]">{formatVND(data.totalExpense)}</p>
          <div className="mt-3">
            <Comparison current={data.totalExpense} previous={data.previousTotalExpense} hasPrevData={data.hasPreviousMonthData} lowerIsBetter />
          </div>
        </div>
        <CategoryStrip legend={data.pie.legend} />
      </div>

      <div className="grid grid-cols-2 gap-3 md:col-span-5 md:grid-cols-1 md:gap-5">
        <StatTile label="Earned" icon={<ArrowDownLeft />} value={`${data.totalIncome > 0 ? '+' : ''}${formatVND(data.totalIncome)}`} valueClass="text-income">
          <Comparison current={data.totalIncome} previous={data.previousTotalIncome} hasPrevData={data.hasPreviousMonthData} />
        </StatTile>
        <StatTile label="Net" icon={<Scale />} value={`${net < 0 ? '−' : net > 0 ? '+' : ''}${formatVND(net)}`} inverted>
          {net < 0 ? 'Spent more than you earned' : 'Earned minus spent'}
        </StatTile>
      </div>
    </section>
  )
}

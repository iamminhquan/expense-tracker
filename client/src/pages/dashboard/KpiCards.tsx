import type { ReactNode } from 'react'
import { ArrowDownLeft, ArrowUpRight, Minus, TrendingDown, TrendingUp, Wallet } from 'lucide-react'
import type { DashboardResponse } from '../../lib/api/types'
import { formatVND } from '../../lib/format'

function Comparison({ current, previous, hasPrevData }: { current: number; previous: number; hasPrevData: boolean }) {
  if (!hasPrevData) return <>No data for last month</>
  const diff = current - previous
  const pct = previous === 0 ? null : Math.round((Math.abs(diff) / previous) * 100)
  const Icon = diff > 0 ? TrendingUp : diff < 0 ? TrendingDown : Minus
  const chip = diff === 0 ? 'unchanged' : pct === null ? (diff > 0 ? 'up' : 'down') : `${diff > 0 ? 'up' : 'down'} ${pct}%`
  return (
    <>
      Last month <span className="tabular">{formatVND(previous)}</span>
      <span className="ml-2 inline-flex h-6 items-center gap-1 rounded-full bg-surface-2 px-2 align-middle text-[12px] font-semibold text-ink">
        <Icon aria-hidden="true" className="size-3.5" />
        {chip}
      </span>
    </>
  )
}

interface KpiProps {
  icon: ReactNode
  iconClass: string
  label: string
  value: string
  valueClass?: string
  children: ReactNode
}

function Kpi({ icon, iconClass, label, value, valueClass = 'text-ink', children }: KpiProps) {
  return (
    <div className="min-w-0 p-5 md:p-7">
      <div className="flex items-center gap-2.5">
        <span aria-hidden="true" className={`flex size-8 items-center justify-center rounded-full [&_svg]:size-[18px] ${iconClass}`}>
          {icon}
        </span>
        <p className="text-[15px] leading-[22px] font-semibold text-ink">{label}</p>
      </div>
      <p
        className={`tabular mt-4 font-display text-[clamp(32px,11vw,46px)] leading-[1.1] font-extrabold tracking-[-0.04em] [overflow-wrap:anywhere] lg:text-[clamp(36px,3.6vw,56px)] ${valueClass}`}
      >
        {value}
      </p>
      <p className="mt-2 text-[14px] leading-5 text-ink-muted">{children}</p>
    </div>
  )
}

export function KpiCards({ data }: { data: DashboardResponse }) {
  const net = data.totalIncome - data.totalExpense
  return (
    <section
      aria-label="Month totals"
      className="grid divide-y divide-border rounded-[24px] border border-border bg-surface md:rounded-[28px] lg:grid-cols-3 lg:divide-x lg:divide-y-0"
    >
      <Kpi icon={<ArrowUpRight />} iconClass="bg-expense-tint text-expense" label="Spent" value={formatVND(data.totalExpense)}>
        <Comparison current={data.totalExpense} previous={data.previousTotalExpense} hasPrevData={data.hasPreviousMonthData} />
      </Kpi>
      <Kpi icon={<ArrowDownLeft />} iconClass="bg-income-tint text-income" label="Earned" value={formatVND(data.totalIncome)}>
        <Comparison current={data.totalIncome} previous={data.previousTotalIncome} hasPrevData={data.hasPreviousMonthData} />
      </Kpi>
      <Kpi
        icon={<Wallet />}
        iconClass="bg-surface-2 text-ink"
        label="Net"
        value={`${net < 0 ? '-' : net > 0 ? '+' : ''}${formatVND(net)}`}
        valueClass={net < 0 ? 'text-danger' : 'text-ink'}
      >
        Earned minus spent in {data.monthLabel}
      </Kpi>
    </section>
  )
}

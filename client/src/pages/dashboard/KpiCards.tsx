import type { ReactNode } from 'react'
import { ArrowDownLeft, ArrowUpRight, Minus, TrendingDown, TrendingUp, Wallet } from 'lucide-react'
import type { DashboardResponse } from '../../lib/api/types'
import { formatVND } from '../../lib/format'

interface Trend {
  Icon: typeof Minus
  text: string
}

function trendOf(current: number, previous: number): Trend {
  const diff = current - previous
  const pct = previous === 0 ? null : Math.round((Math.abs(diff) / previous) * 100)
  const Icon = diff > 0 ? TrendingUp : diff < 0 ? TrendingDown : Minus
  const text = diff === 0 ? 'unchanged' : pct === null ? (diff > 0 ? 'up' : 'down') : `${diff > 0 ? 'up' : 'down'} ${pct}%`
  return { Icon, text }
}

function Comparison({ current, previous, hasPrevData, onAccent }: { current: number; previous: number; hasPrevData: boolean; onAccent?: boolean }) {
  if (!hasPrevData) return <span>No data for last month</span>
  const { Icon, text } = trendOf(current, previous)
  return (
    <span className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
      <span
        className={`inline-flex h-6 items-center gap-1 rounded-full px-2 text-[12px] leading-4 font-semibold ${onAccent ? 'bg-on-accent text-accent' : 'bg-surface text-ink'}`}
      >
        <Icon aria-hidden="true" className="size-3.5" />
        {text}
      </span>
      <span>
        Last month <span className="tabular">{formatVND(previous)}</span>
      </span>
    </span>
  )
}

function IconTile({ children, className }: { children: ReactNode; className: string }) {
  return (
    <span aria-hidden="true" className={`flex size-10 shrink-0 items-center justify-center rounded-tile [&_svg]:size-5 ${className}`}>
      {children}
    </span>
  )
}

const smallValueClass =
  'num text-[clamp(22px,6.2vw,30px)] leading-[1.1] font-bold tracking-[-0.03em] [overflow-wrap:anywhere] sm:mt-1 lg:text-[clamp(26px,2.4vw,36px)]'

/* A slim row on phones, where two columns would wrap the amounts; a square-ish tile from sm up. */
const smallTileClass = 'col-span-2 flex min-w-0 items-center gap-4 p-4 sm:col-span-1 sm:flex-col sm:items-stretch sm:gap-0 md:p-6 lg:col-span-3'

function SpentHero({ data }: { data: DashboardResponse }) {
  const ofIncome = data.totalIncome > 0 ? Math.round((data.totalExpense / data.totalIncome) * 100) : null
  return (
    <div className="relative col-span-2 overflow-hidden rounded-hero bg-accent p-6 text-on-accent shadow-card md:p-8 lg:col-span-6">
      <svg viewBox="0 0 400 400" aria-hidden="true" className="pointer-events-none absolute overflow-visible -top-24 -right-20 size-[340px] text-on-accent opacity-[0.1]" fill="none" stroke="currentColor" strokeWidth="26">
        <circle cx="200" cy="200" r="64" />
        <circle cx="200" cy="200" r="132" />
        <circle cx="200" cy="200" r="200" />
      </svg>
      <div className="relative flex items-center gap-3">
        <IconTile className="bg-on-accent/15">
          <ArrowUpRight />
        </IconTile>
        <p className="text-[15px] leading-[22px] font-semibold">Spent in {data.monthLabel}</p>
      </div>
      <p className="num relative mt-7 text-[clamp(44px,14vw,68px)] leading-none font-bold tracking-[-0.045em] [overflow-wrap:anywhere] md:mt-9 lg:text-[clamp(48px,4.6vw,72px)]">
        <span className="sr-only">Spent: </span>
        {formatVND(data.totalExpense)}
      </p>
      <div className="relative mt-6 space-y-4 text-[14px] leading-5 text-on-accent-muted md:mt-8">
        {ofIncome !== null && (
          <div>
            <div role="img" aria-label={`Spent ${ofIncome}% of what you earned`} className="h-2 overflow-hidden rounded-full bg-on-accent/20">
              <div className="h-full rounded-full bg-on-accent transition-[width] duration-500" style={{ width: `${Math.min(100, ofIncome)}%` }} />
            </div>
            <p className="mt-2">{ofIncome}% of what you earned</p>
          </div>
        )}
        <Comparison current={data.totalExpense} previous={data.previousTotalExpense} hasPrevData={data.hasPreviousMonthData} onAccent />
      </div>
    </div>
  )
}

function EarnedTile({ data }: { data: DashboardResponse }) {
  return (
    <div className={`${smallTileClass} rounded-card bg-income-tint text-ink`}>
      <IconTile className="bg-surface text-income">
        <ArrowDownLeft />
      </IconTile>
      <div className="min-w-0 flex-1 sm:mt-4 sm:flex sm:flex-col">
        <p className="text-[14px] leading-5 font-semibold text-ink">Earned</p>
        <p className={`${smallValueClass} text-income`}>
          <span className="sr-only">Earned: </span>
          {data.totalIncome > 0 ? '+' : ''}
          {formatVND(data.totalIncome)}
        </p>
        <p className="mt-1.5 text-[13px] leading-[18px] text-ink-muted sm:mt-auto sm:pt-4">
          <Comparison current={data.totalIncome} previous={data.previousTotalIncome} hasPrevData={data.hasPreviousMonthData} />
        </p>
      </div>
    </div>
  )
}

function NetTile({ data }: { data: DashboardResponse }) {
  const net = data.totalIncome - data.totalExpense
  return (
    <div className={`${smallTileClass} rounded-card border border-border bg-surface shadow-card`}>
      <IconTile className="bg-surface-2 text-ink">
        <Wallet />
      </IconTile>
      <div className="min-w-0 flex-1 sm:mt-4 sm:flex sm:flex-col">
        <p className="text-[14px] leading-5 font-semibold text-ink">Net</p>
        <p className={`${smallValueClass} ${net < 0 ? 'text-danger' : 'text-ink'}`}>
          <span className="sr-only">Net: </span>
          {net < 0 ? '−' : net > 0 ? '+' : ''}
          {formatVND(net)}
        </p>
        <p className="mt-1.5 text-[13px] leading-[18px] text-ink-muted sm:mt-auto sm:pt-4">Earned minus spent in {data.monthLabel}</p>
      </div>
    </div>
  )
}

export function KpiCards({ data }: { data: DashboardResponse }) {
  return (
    <section aria-label="Month totals" className="grid grid-cols-2 gap-3 md:gap-5 lg:grid-cols-12">
      <SpentHero data={data} />
      <EarnedTile data={data} />
      <NetTile data={data} />
    </section>
  )
}

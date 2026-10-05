import { useState } from 'react'
import { Bar, Doughnut } from 'react-chartjs-2'
import '../lib/charts'
import { useDashboard } from '../hooks/useDashboard'
import { MonthPicker } from '../components/MonthPicker'
import { formatVND, formatVNDSigned } from '../lib/format'

function comparison(current: number, previous: number, hasPrevData: boolean): string {
  if (!hasPrevData) return 'No data for last month'
  if (previous === 0) return `Last month ${formatVND(previous)}`
  const diff = current - previous
  const pct = Math.round((Math.abs(diff) / previous) * 100)
  if (diff === 0) return `Last month ${formatVND(previous)} · unchanged`
  return `Last month ${formatVND(previous)} · ${diff > 0 ? 'up' : 'down'} ${pct}%`
}

export function DashboardPage() {
  const [month, setMonth] = useState<string | undefined>(undefined)
  const { data, isLoading, error } = useDashboard(month)

  if (isLoading || !data) return <p className="text-ink-faint">Loading…</p>
  if (error) return <p className="text-expense">Could not load the dashboard.</p>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-[20px] font-semibold">Overview</h1>
        <MonthPicker
          value={data.monthValue}
          label={data.monthLabel}
          currentMonthValue={data.currentMonthValue}
          availableMonths={data.availableMonths}
          onChange={setMonth}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="rounded-[16px] border border-border-card bg-surface p-5">
          <p className="mb-1 text-[12px] font-semibold uppercase tracking-wide text-ink-faint">Spent</p>
          <p className="font-mono text-[26px] font-semibold text-expense">{formatVND(data.totalExpense)}</p>
          <p className="mt-1 text-[13px] text-ink-faint">{comparison(data.totalExpense, data.previousTotalExpense, data.hasPreviousMonthData)}</p>
        </div>
        <div className="rounded-[16px] border border-border-card bg-surface p-5">
          <p className="mb-1 text-[12px] font-semibold uppercase tracking-wide text-ink-faint">Earned</p>
          <p className="font-mono text-[26px] font-semibold text-income">{formatVND(data.totalIncome)}</p>
          <p className="mt-1 text-[13px] text-ink-faint">{comparison(data.totalIncome, data.previousTotalIncome, data.hasPreviousMonthData)}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="rounded-[16px] border border-border-card bg-surface p-5">
          <p className="mb-4 text-[14px] font-semibold">Spending by category</p>
          {data.pie.labels.length === 0 ? (
            <p className="text-[13px] text-ink-faint">No expenses this month.</p>
          ) : (
            <div className="flex flex-col items-center gap-4 sm:flex-row">
              <div className="h-[180px] w-[180px] shrink-0">
                <Doughnut
                  data={{
                    labels: data.pie.labels,
                    datasets: [{ data: data.pie.values, backgroundColor: data.pie.colors, borderWidth: 0 }],
                  }}
                  options={{ plugins: { legend: { display: false } }, cutout: '65%' }}
                />
              </div>
              <ul className="w-full space-y-1.5">
                {data.pie.legend.map((entry) => (
                  <li key={entry.name} className="flex items-center justify-between text-[13px]">
                    <span className="flex items-center gap-2">
                      <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: entry.color }} />
                      <span className="text-ink-muted">{entry.name}</span>
                    </span>
                    <span className="font-mono text-ink-faint">
                      {entry.percent}% · {formatVND(entry.amount)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <div className="rounded-[16px] border border-border-card bg-surface p-5">
          <p className="mb-4 text-[14px] font-semibold">Last {data.bar.labels.length} months</p>
          <Bar
            data={{
              labels: data.bar.labels,
              datasets: [
                { label: 'Expense', data: data.bar.expense, backgroundColor: 'rgb(180 35 24)' },
                { label: 'Income', data: data.bar.income, backgroundColor: 'rgb(47 125 91)' },
              ],
            }}
            options={{ responsive: true, plugins: { legend: { position: 'bottom' } }, scales: { y: { beginAtZero: true } } }}
          />
        </div>
      </div>

      {!data.headerBalance.empty && (
        <p className="text-center text-[13px] text-ink-faint">
          Balance carried into this month: <span className="font-mono">{formatVNDSigned(data.headerBalance.remaining)}</span>
        </p>
      )}
    </div>
  )
}

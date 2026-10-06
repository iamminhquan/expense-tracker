import { Bar } from 'react-chartjs-2'
import '../../lib/charts'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { useThemeColors } from '../../hooks/useThemeColors'
import { cardClass, cardTitleClass } from '../../lib/formStyles'
import { formatVND } from '../../lib/format'
import type { BarData } from '../../lib/api/types'

const STEP = 5_000_000

export function MonthlyBars({ bar }: { bar: BarData }) {
  const colors = useThemeColors()
  const isDesktop = useIsDesktop()
  // A theme switch remounts the chart (see useThemeColors); only the first draw animates.
  const animate = !colors.switched && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const peak = Math.max(0, ...bar.expense, ...bar.income)
  const suggestedMax = Math.max(STEP, Math.ceil(peak / STEP) * STEP)
  const lastIndex = bar.labels.length - 1
  const radius = { topLeft: 10, topRight: 10, bottomLeft: 0, bottomRight: 0 }

  return (
    <section aria-labelledby="bars-title" className={cardClass}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="bars-title" className={cardTitleClass}>
          Last {bar.labels.length} months
        </h2>
        <ul className="flex items-center gap-4 text-[13px] leading-[18px] text-ink-muted" aria-hidden="true">
          <li className="flex items-center gap-2">
            <span className="size-3.5 rounded-[5px] bg-expense" />
            Expense
          </li>
          <li className="flex items-center gap-2">
            <span className="size-3.5 rounded-[5px] bg-chart-income" />
            Income
          </li>
        </ul>
      </div>
      <div className="mt-6" style={{ height: isDesktop ? 300 : 250 }}>
        <Bar
          key={colors.key}
          role="img"
          aria-label={`Expense and income by month: ${bar.labels
            .map((m, i) => `${m} expense ${formatVND(bar.expense[i])}, income ${formatVND(bar.income[i])}`)
            .join('; ')}`}
          data={{
            labels: bar.labels,
            datasets: [
              { label: 'Expense', data: bar.expense, backgroundColor: colors.expense, borderRadius: radius, borderSkipped: false },
              { label: 'Income', data: bar.income, backgroundColor: colors.income, borderRadius: radius, borderSkipped: false },
            ],
          }}
          options={{
            maintainAspectRatio: false,
            animation: animate ? { duration: 400, easing: 'easeOutQuart' } : false,
            datasets: { bar: { categoryPercentage: isDesktop ? 0.7 : 0.85, barPercentage: 0.9, maxBarThickness: 36, borderWidth: 0 } },
            layout: { padding: { top: 8, right: 4 } },
            interaction: { mode: 'index', intersect: false },
            plugins: {
              legend: { display: false },
              tooltip: {
                backgroundColor: colors.surface,
                borderColor: colors.grid,
                borderWidth: 1,
                titleColor: colors.ink,
                bodyColor: colors.tick,
                padding: 12,
                cornerRadius: 12,
                boxPadding: 6,
                callbacks: { label: (item) => `${item.dataset.label}: ${formatVND(Number(item.raw))}` },
              },
            },
            scales: {
              x: {
                border: { display: false },
                grid: { display: false },
                ticks: {
                  padding: 8,
                  color: (ctx) => (ctx.index === lastIndex ? colors.ink : colors.tick),
                  font: (ctx) => ({ weight: ctx.index === lastIndex ? 600 : 400 }),
                },
              },
              y: {
                beginAtZero: true,
                suggestedMax,
                border: { display: false },
                grid: { color: colors.grid },
                ticks: {
                  stepSize: STEP,
                  padding: 8,
                  color: colors.tick,
                  callback: (v) => (v === 0 ? '0' : `${Number(v) / 1e6}M`),
                },
              },
            },
          }}
        />
      </div>
      <div className="sr-only">
        <table>
          <caption>Expense and income by month</caption>
          <thead>
            <tr>
              <th scope="col">Month</th>
              <th scope="col">Expense</th>
              <th scope="col">Income</th>
            </tr>
          </thead>
          <tbody>
            {bar.labels.map((m, i) => (
              <tr key={m}>
                <th scope="row">{m}</th>
                <td>{formatVND(bar.expense[i])}</td>
                <td>{formatVND(bar.income[i])}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

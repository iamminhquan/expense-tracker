import { Doughnut } from 'react-chartjs-2'
import { Receipt } from 'lucide-react'
import '../../lib/charts'
import { EmptyState } from '../../components/ui/EmptyState'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { useThemeColors } from '../../hooks/useThemeColors'
import { cardClass, cardTitleClass } from '../../lib/formStyles'
import { formatVND } from '../../lib/format'
import type { PieData } from '../../lib/api/types'

interface SpendingDoughnutProps {
  pie: PieData
  total: number
  monthLabel: string
}

export function SpendingDoughnut({ pie, total, monthLabel }: SpendingDoughnutProps) {
  const colors = useThemeColors()
  const isDesktop = useIsDesktop()
  // A theme switch remounts the chart (see useThemeColors); only the first draw animates.
  const animate = !colors.switched && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const size = isDesktop ? 208 : 232

  return (
    <section aria-labelledby="spending-title" className={`${cardClass} h-full`}>
      <h2 id="spending-title" className={cardTitleClass}>
        Spending by category
      </h2>
      {pie.legend.length === 0 ? (
        <EmptyState icon={<Receipt />} title="No expenses this month">
          Nothing was spent in {monthLabel}, so there's nothing to break down yet.
        </EmptyState>
      ) : (
        <div className="mt-6 flex flex-col items-center gap-7 sm:flex-row sm:items-center lg:flex-col lg:items-stretch">
          <div className="relative shrink-0 self-center" style={{ width: size, height: size }}>
            <Doughnut
              key={colors.key}
              role="img"
              aria-label={`Spending by category: ${pie.legend.map((e) => `${e.name} ${e.percent}%`).join(', ')}`}
              data={{
                labels: pie.labels,
                datasets: [
                  {
                    data: pie.values,
                    backgroundColor: pie.colors,
                    borderWidth: 0,
                    spacing: 3,
                    borderRadius: 8,
                    hoverOffset: 6,
                  },
                ],
              }}
              options={{
                cutout: '72%',
                layout: { padding: 6 },
                maintainAspectRatio: true,
                animation: animate ? { duration: 400, easing: 'easeOutQuart' } : false,
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
                    callbacks: { label: (item) => `${item.label}: ${formatVND(Number(item.raw))}` },
                  },
                },
              }}
            />
            <div aria-hidden="true" className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-[12px] leading-4 text-ink-muted">Spent</span>
              <span className="num text-[19px] leading-6 font-bold text-ink md:text-[20px]">{formatVND(total)}</span>
            </div>
          </div>
          <ul className="w-full min-w-0 divide-y divide-border">
            {pie.legend.map((entry) => (
              <li key={entry.name} className="flex items-center gap-3 py-2.5 text-[14px] leading-5">
                <span aria-hidden="true" className="size-3 shrink-0 rounded-[4px]" style={{ backgroundColor: entry.color }} />
                <span className="min-w-0 flex-1 truncate font-semibold text-ink">{entry.name}</span>
                <span className="tabular w-10 text-right text-[13px] text-ink-muted">{entry.percent}%</span>
                <span className="num w-[104px] text-right font-bold text-ink">{formatVND(entry.amount)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}

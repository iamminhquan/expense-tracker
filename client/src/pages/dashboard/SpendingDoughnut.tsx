import { useState } from 'react'
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
  const [firstKey] = useState(colors.key)
  const animate = colors.key === firstKey && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const size = isDesktop ? 200 : 224

  return (
    <section aria-labelledby="spending-title" className={cardClass}>
      <h2 id="spending-title" className={cardTitleClass}>
        Spending by category
      </h2>
      {pie.legend.length === 0 ? (
        <EmptyState icon={<Receipt />} title="No expenses this month">
          Nothing was spent in {monthLabel}, so there's nothing to break down yet.
        </EmptyState>
      ) : (
        <div className="mt-6 flex flex-col items-center gap-6 sm:flex-row sm:items-center">
          <div className="relative shrink-0" style={{ width: size, height: size }}>
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
                    borderWidth: 4,
                    borderColor: colors.surface,
                    hoverBorderColor: colors.surface,
                    hoverOffset: 0,
                  },
                ],
              }}
              options={{
                cutout: '68%',
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
              <span className="tabular font-display text-[17px] leading-6 font-extrabold text-ink md:text-[19px]">{formatVND(total)}</span>
            </div>
          </div>
          <ul className="w-full min-w-0 space-y-2.5">
            {pie.legend.map((entry) => (
              <li key={entry.name} className="flex items-center gap-3 text-[13px] leading-[18px]">
                <span aria-hidden="true" className="size-3.5 shrink-0 rounded-[5px]" style={{ backgroundColor: entry.color }} />
                <span className="min-w-0 flex-1 truncate font-semibold text-ink">{entry.name}</span>
                <span className="tabular w-10 text-right text-ink-muted">{entry.percent}%</span>
                <span className="tabular w-[104px] text-right font-semibold text-ink">{formatVND(entry.amount)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}

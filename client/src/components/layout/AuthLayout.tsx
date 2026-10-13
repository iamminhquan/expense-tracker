import { useState, type ReactNode } from 'react'
import { Wordmark } from '../ui/Wordmark'

// Made-up daily totals as a share of the tallest day; the 1st carries the rent.
function sampleShare(date: Date) {
  const day = date.getDate()
  if (day === 1) return 1
  const noise = Math.abs(Math.sin(day * 91.7) * 1000) % 1
  const weekend = date.getDay() === 0 || date.getDay() === 6
  return 0.1 + noise * 0.45 + (weekend ? 0.22 : 0)
}

/* Decorative: the current month drawn one bar a day, the days after today fainter. */
function MonthStrip() {
  const [now] = useState(() => new Date())
  const today = now.getDate()
  const daysInMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate()
  const month = now.toLocaleDateString('en-GB', { month: 'short' })
  const days = Array.from({ length: daysInMonth }, (_, i) => new Date(now.getFullYear(), now.getMonth(), i + 1))

  return (
    <div>
      <div className="flex h-[clamp(140px,26vh,260px)] items-end gap-[3px] border-b border-on-accent/25">
        {days.map((date) => {
          const day = date.getDate()
          return (
            <div key={day} className="flex h-full flex-1 items-end justify-center">
              <span
                style={{ height: `${sampleShare(date) * 100}%`, animationDelay: `${day * 18}ms` }}
                className={`block w-full max-w-[14px] origin-bottom animate-grow-bar rounded-t-full ${
                  day === today ? 'bg-on-accent' : day < today ? 'bg-on-accent/30' : 'bg-on-accent/12'
                }`}
              />
            </div>
          )
        })}
      </div>
      <div className="relative mt-2.5 h-4 text-[12px] leading-4 text-on-accent-muted tabular">
        {today > 4 && <span className="absolute left-0">1 {month}</span>}
        <span
          className="absolute -translate-x-1/2 font-semibold text-on-accent"
          style={{ left: `${((today - 0.5) / daysInMonth) * 100}%` }}
        >
          Today
        </span>
        {today < daysInMonth - 3 && (
          <span className="absolute right-0">
            {daysInMonth} {month}
          </span>
        )}
      </div>
    </div>
  )
}

function BrandPanel() {
  return (
    <aside aria-hidden="true" className="hidden w-[46%] max-w-[640px] shrink-0 flex-col justify-between rounded-hero bg-accent p-12 text-on-accent lg:flex">
      <Wordmark onAccent className="text-[36px] leading-10" />
      <div>
        <p className="max-w-[420px] font-display text-[46px] leading-[50px] font-bold tracking-[-0.04em]">See where your money goes.</p>
        <p className="mt-4 max-w-[380px] text-[16px] leading-6 text-on-accent-muted">Log a spend in a few taps and watch the month take shape.</p>
        <div className="mt-12">
          <MonthStrip />
        </div>
      </div>
    </aside>
  )
}

/*
 * On desktop the page is exactly one viewport tall and only the form column scrolls, so the
 * panel keeps its size whichever form is showing: a document scrollbar would narrow it.
 */
export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh bg-app lg:h-dvh lg:overflow-hidden lg:p-4">
      <BrandPanel />
      <div className="flex min-w-0 flex-1 flex-col items-center px-4 pt-8 pb-10 sm:px-6 sm:pt-16 lg:overflow-y-auto lg:py-4">
        <p className="mb-7 lg:hidden">
          <Wordmark className="text-[34px] leading-10" />
        </p>
        <main className="w-full max-w-[440px] animate-page-in space-y-5 rounded-hero lg:space-y-3 border border-border bg-surface p-6 shadow-card sm:p-9 lg:my-auto lg:border-0 lg:bg-transparent lg:p-0 lg:shadow-none">
          {children}
        </main>
      </div>
    </div>
  )
}

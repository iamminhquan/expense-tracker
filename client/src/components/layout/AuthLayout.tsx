import type { ReactNode } from 'react'
import { Wordmark } from '../ui/Wordmark'

const SAMPLE = [
  { name: 'Cà phê sáng', meta: 'Food · Today', amount: '−45.000₫', income: false },
  { name: 'Tiền nhà', meta: 'Rent · 01 Oct', amount: '−3.800.000₫', income: false },
  { name: 'Lương tháng 10', meta: 'Salary · 01 Oct', amount: '+18.500.000₫', income: true },
]

/* Decorative: a few made-up rows so the panel shows what the app feels like. */
function BrandPanel() {
  return (
    <aside aria-hidden="true" className="relative hidden w-[46%] max-w-[640px] shrink-0 flex-col justify-between overflow-hidden rounded-hero bg-accent p-12 text-on-accent lg:flex">
      <svg viewBox="0 0 400 400" className="absolute overflow-visible -right-24 -bottom-24 size-[460px] text-on-accent opacity-[0.09]" fill="none" stroke="currentColor" strokeWidth="28">
        <circle cx="200" cy="200" r="60" />
        <circle cx="200" cy="200" r="130" />
        <circle cx="200" cy="200" r="200" />
      </svg>
      <Wordmark onAccent className="relative text-[36px] leading-10" />
      <div className="relative">
        <p className="max-w-[420px] font-display text-[46px] leading-[50px] font-bold tracking-[-0.04em]">See where your money goes.</p>
        <p className="mt-4 max-w-[380px] text-[16px] leading-6 text-on-accent-muted">Log a spend in a few taps and watch the month take shape.</p>
        <ul className="mt-10 max-w-[420px] space-y-2.5">
          {SAMPLE.map((row, i) => (
            <li
              key={row.name}
              style={{ animationDelay: `${i * 90}ms` }}
              className="flex animate-rise items-center gap-3 rounded-[18px] bg-surface/95 px-4 py-3 text-ink shadow-popover"
            >
              <span className={`flex size-10 items-center justify-center rounded-tile font-display text-[16px] font-bold ${row.income ? 'bg-income-tint text-income' : 'bg-accent-tint text-accent-text'}`}>
                {row.name[0]}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-[14px] leading-5 font-semibold">{row.name}</span>
                <span className="block text-[12px] leading-4 text-ink-muted">{row.meta}</span>
              </span>
              <span className={`num text-[15px] font-bold ${row.income ? 'text-income' : 'text-ink'}`}>{row.amount}</span>
            </li>
          ))}
        </ul>
      </div>
      <span />
    </aside>
  )
}

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen bg-app lg:p-4">
      <BrandPanel />
      <div className="flex min-w-0 flex-1 flex-col items-center px-4 pt-8 pb-10 sm:px-6 sm:pt-16 lg:justify-center lg:pt-8">
        <p className="mb-7 lg:hidden">
          <Wordmark className="text-[34px] leading-10" />
        </p>
        <main className="w-full max-w-[440px] animate-page-in space-y-5 rounded-hero border border-border bg-surface p-6 shadow-card sm:p-9 lg:border-0 lg:bg-transparent lg:p-0 lg:shadow-none">
          {children}
        </main>
      </div>
    </div>
  )
}

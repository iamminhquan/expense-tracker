import type { ReactNode } from 'react'
import { SWATCHES } from '../../lib/categorySwatches'
import { Wordmark } from '../ui/Wordmark'

const SAMPLE = [
  { note: 'Cà phê sữa đá', category: 'Food & drink', amount: '−29.000₫', swatch: SWATCHES[0] },
  { note: 'Grab to work', category: 'Transport', amount: '−64.000₫', swatch: SWATCHES[1] },
  { note: 'October salary', category: 'Salary', amount: '+18.500.000₫', swatch: SWATCHES[6], income: true },
]

// A drawing of the app, not data: made-up rows painted in the real category swatches.
function ReceiptArt() {
  return (
    <div aria-hidden="true" className="relative mx-auto w-full max-w-[360px]">
      <div className="absolute inset-x-6 -top-4 h-full rotate-[-4deg] rounded-[26px] bg-on-brand/15" />
      <div className="relative rounded-[26px] bg-surface p-5 text-ink shadow-popover">
        <div className="flex items-baseline justify-between">
          <p className="text-[13px] font-medium text-ink-muted">Spent in October</p>
          <p className="text-[12px] font-semibold text-income">−8% vs Sep</p>
        </div>
        <p className="figure mt-1 text-[46px] leading-none">4.862.000₫</p>
        <div className="mt-4 flex h-2.5 gap-0.5 overflow-hidden rounded-full">
          {[38, 24, 18, 12].map((share, i) => (
            <span key={share} style={{ width: `${share}%`, backgroundColor: SWATCHES[[0, 1, 2, 4][i]] }} />
          ))}
          <span className="flex-1 bg-border" />
        </div>
        <ul className="mt-5 space-y-3.5">
          {SAMPLE.map((row) => (
            <li key={row.note} className="flex items-center gap-3">
              <span className="size-9 shrink-0 rounded-full" style={{ backgroundColor: row.swatch }} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-[14px] leading-5 font-semibold">{row.note}</p>
                <p className="text-[12px] leading-4 text-ink-muted">{row.category}</p>
              </div>
              <p className={`figure-sm text-[16px] ${row.income ? 'text-income' : ''}`}>{row.amount}</p>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-app lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <aside className="relative hidden overflow-hidden bg-brand px-12 py-12 text-on-brand lg:flex lg:flex-col xl:px-16">
        <span role="img" aria-label="$pend" className="wordmark text-[30px] text-on-brand">
          $pend
        </span>
        <div className="my-auto py-12">
          <p className="heading max-w-[460px] text-[44px] leading-[48px] xl:text-[52px] xl:leading-[56px]">Know where every đồng goes.</p>
          <p className="mt-4 max-w-[400px] text-[17px] leading-[26px] opacity-90">Log a spend in a few taps on your phone, then see the whole month at a glance.</p>
          <div className="mt-14">
            <ReceiptArt />
          </div>
        </div>
      </aside>

      <div className="flex min-h-screen flex-col items-center px-4 pt-10 pb-12 sm:px-6 sm:pt-16 lg:justify-center lg:py-16">
        <div className="mb-8 w-full max-w-[420px] lg:hidden">
          <Wordmark className="text-[32px]" />
          <p className="mt-2 text-[15px] leading-[22px] text-ink-muted">Know where every đồng goes.</p>
        </div>
        <main className="w-full max-w-[420px] animate-page-in space-y-6 rounded-tile border border-border bg-surface p-6 sm:p-8">{children}</main>
      </div>
    </div>
  )
}

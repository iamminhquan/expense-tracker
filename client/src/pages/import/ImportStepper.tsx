import { Check } from 'lucide-react'

const STEPS = ['Upload', 'Map columns', 'Preview', 'Done']

export function ImportStepper({ current }: { current: number }) {
  return (
    <>
      <ol className="hidden items-center gap-3 md:flex" aria-label="Import progress">
        {STEPS.map((label, i) => {
          const done = i < current
          const active = i === current
          return (
            <li key={label} aria-current={active ? 'step' : undefined} className="flex flex-1 items-center gap-3 last:flex-none">
              <span
                className={`tabular flex size-8 shrink-0 items-center justify-center rounded-full text-[14px] font-bold ${
                  done || active ? 'bg-accent text-on-accent' : 'border border-border-strong text-ink-muted'
                }`}
              >
                {done ? <Check aria-hidden="true" className="size-4" /> : i + 1}
              </span>
              <span className={`text-[14px] leading-5 whitespace-nowrap ${active ? 'font-bold text-ink' : 'text-ink-muted'}`}>
                {label}
                {done && <span className="sr-only"> (done)</span>}
              </span>
              {i < STEPS.length - 1 && <span aria-hidden="true" className={`h-0.5 flex-1 rounded-full ${done ? 'bg-accent' : 'bg-border'}`} />}
            </li>
          )
        })}
      </ol>
      <div className="md:hidden">
        <p className="text-[13px] leading-[18px] text-ink-muted">
          Step {current + 1} of {STEPS.length}
        </p>
        <p className="font-display text-[20px] leading-6 font-semibold tracking-[-0.015em] text-ink">{STEPS[current]}</p>
        <div aria-hidden="true" className="mt-2 grid grid-cols-4 gap-1.5">
          {STEPS.map((label, i) => (
            <span key={label} className={`h-1.5 rounded-full ${i <= current ? 'bg-accent' : 'bg-surface-2'}`} />
          ))}
        </div>
      </div>
    </>
  )
}

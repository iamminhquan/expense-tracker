import type { ReactNode } from 'react'

export type BadgeKind = 'warning' | 'neutral' | 'expense' | 'income'

const kinds: Record<BadgeKind, string> = {
  warning: 'bg-warning-tint text-warning',
  neutral: 'bg-surface-2 text-ink-muted',
  expense: 'bg-expense-tint text-expense',
  income: 'bg-income-tint text-income',
}

export function Badge({ kind = 'neutral', icon, children }: { kind?: BadgeKind; icon?: ReactNode; children: ReactNode }) {
  return (
    <span
      className={`inline-flex h-6 shrink-0 items-center gap-1 rounded-full pr-2.5 pl-2 text-[12px] leading-4 font-semibold whitespace-nowrap [&_svg]:size-3.5 ${kinds[kind]}`}
    >
      {icon}
      {children}
    </span>
  )
}

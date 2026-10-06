import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon: ReactNode
  title: string
  children?: ReactNode
  actions?: ReactNode
}

export function EmptyState({ icon, title, children, actions }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center px-6 py-12 text-center">
      <span className="mb-4 flex size-16 items-center justify-center rounded-full bg-surface-2 text-ink [&_svg]:size-7" aria-hidden="true">
        {icon}
      </span>
      <p className="font-display text-[20px] leading-[26px] font-bold text-ink">{title}</p>
      {children && <div className="mt-1.5 max-w-[360px] text-[14px] leading-[22px] text-ink-muted">{children}</div>}
      {actions && <div className="mt-5 flex flex-wrap justify-center gap-3">{actions}</div>}
    </div>
  )
}

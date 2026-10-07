import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon: ReactNode
  title: string
  children?: ReactNode
  actions?: ReactNode
}

/* Inline art, so there is no image to fetch; swap in an illustration here later. */
function EmptyArt({ icon }: { icon: ReactNode }) {
  return (
    <div aria-hidden="true" className="relative mb-5 size-[132px]">
      <svg viewBox="0 0 132 132" className="absolute inset-0 size-full" fill="none">
        <circle cx="66" cy="66" r="62" className="stroke-border" strokeWidth="1.5" strokeDasharray="3 7" strokeLinecap="round" />
        <circle cx="66" cy="66" r="44" className="fill-accent-tint" />
        <rect x="14" y="26" width="22" height="22" rx="7" className="fill-income-tint" transform="rotate(-12 25 37)" />
        <circle cx="112" cy="40" r="7" className="fill-surface-2" />
        <circle cx="104" cy="108" r="4" className="fill-accent-tint" />
        <rect x="22" y="94" width="14" height="14" rx="5" className="fill-surface-2" transform="rotate(14 29 101)" />
      </svg>
      <span className="absolute top-1/2 left-1/2 flex size-[68px] -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-[22px] border border-border bg-surface text-accent-text shadow-card [&_svg]:size-8">
        {icon}
      </span>
    </div>
  )
}

export function EmptyState({ icon, title, children, actions }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center px-6 py-10 text-center md:py-14">
      <EmptyArt icon={icon} />
      <p className="font-display text-[22px] leading-7 font-semibold tracking-[-0.015em] text-ink">{title}</p>
      {children && <div className="mt-1.5 max-w-[360px] text-[14px] leading-[22px] text-ink-muted">{children}</div>}
      {actions && <div className="mt-5 flex flex-wrap justify-center gap-3">{actions}</div>}
    </div>
  )
}

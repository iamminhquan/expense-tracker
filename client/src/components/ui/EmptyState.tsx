import type { ReactNode } from 'react'
import { Illustration, type IllustrationName } from './Illustration'

interface EmptyStateProps {
  art: IllustrationName
  title: string
  children?: ReactNode
  actions?: ReactNode
  /** Tighter spacing for an empty state inside a dashboard tile. */
  compact?: boolean
}

export function EmptyState({ art, title, children, actions, compact }: EmptyStateProps) {
  return (
    <div className={`flex animate-fade-in flex-col items-center text-center ${compact ? 'px-2 py-6' : 'px-6 py-12'}`}>
      <Illustration name={art} className={compact ? 'mb-3 h-[84px]' : 'mb-5 h-[112px]'} />
      <p className="heading text-[19px] leading-[26px] text-ink">{title}</p>
      {children && <div className="mt-1.5 max-w-[340px] text-[14px] leading-[21px] text-ink-muted">{children}</div>}
      {actions && <div className="mt-6 flex flex-wrap justify-center gap-3">{actions}</div>}
    </div>
  )
}

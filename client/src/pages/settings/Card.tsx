import { useId, type ReactNode } from 'react'
import { cardClass } from '../../lib/formStyles'

interface CardProps {
  title: string
  description?: ReactNode
  danger?: boolean
  className?: string
  children: ReactNode
}

export function Card({ title, description, danger, className = '', children }: CardProps) {
  const titleId = useId()
  return (
    <section aria-labelledby={titleId} className={`${cardClass} space-y-[18px] ${danger ? 'border-danger' : ''} ${className}`}>
      <div>
        <h2 id={titleId} className={`font-display text-[20px] leading-[26px] font-bold ${danger ? 'text-danger' : 'text-ink'}`}>
          {title}
        </h2>
        {description && <p className="mt-1 text-[14px] leading-5 text-ink-muted">{description}</p>}
      </div>
      {children}
    </section>
  )
}

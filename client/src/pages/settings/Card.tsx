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
    <section aria-labelledby={titleId} className={`${cardClass} space-y-5 ${danger ? 'border-danger/45' : ''} ${className}`}>
      <div>
        <h2 id={titleId} className={`heading text-[19px] leading-[26px] ${danger ? 'text-danger' : 'text-ink'}`}>
          {title}
        </h2>
        {description && <p className="mt-1 text-[14px] leading-[21px] text-ink-muted">{description}</p>}
      </div>
      {children}
    </section>
  )
}

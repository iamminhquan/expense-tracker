import type { SelectHTMLAttributes } from 'react'
import { ChevronDown } from 'lucide-react'
import { selectClass } from '../../lib/formStyles'

export function SelectControl({ className = '', children, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className="relative">
      <select {...props} className={`${selectClass} ${className}`}>
        {children}
      </select>
      <ChevronDown aria-hidden="true" className="pointer-events-none absolute top-1/2 right-3 size-[18px] -translate-y-1/2 text-ink-muted" />
    </div>
  )
}

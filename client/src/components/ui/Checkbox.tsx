import type { ReactNode } from 'react'
import { Check } from 'lucide-react'

interface CheckboxProps {
  checked: boolean
  onChange: (checked: boolean) => void
  children: ReactNode
}

export function Checkbox({ checked, onChange, children }: CheckboxProps) {
  return (
    <label className="flex min-h-11 cursor-pointer items-center gap-3 text-[15px] leading-[22px] font-medium text-ink">
      <span className="relative flex size-6 shrink-0">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="peer absolute inset-0 m-0 cursor-pointer appearance-none rounded-[8px] border-[1.5px] border-border-strong bg-surface checked:border-accent checked:bg-accent"
        />
        <Check aria-hidden="true" strokeWidth={3} className="pointer-events-none relative m-auto size-4 scale-50 text-on-accent opacity-0 transition-transform peer-checked:scale-100 peer-checked:opacity-100" />
      </span>
      {children}
    </label>
  )
}

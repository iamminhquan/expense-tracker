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
      <span className="relative flex size-[22px] shrink-0">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="peer absolute inset-0 m-0 cursor-pointer appearance-none rounded-[6px] border-[1.5px] border-border-strong bg-surface checked:border-accent checked:bg-accent"
        />
        <Check aria-hidden="true" strokeWidth={3} className="pointer-events-none relative m-auto size-4 text-on-accent opacity-0 peer-checked:opacity-100" />
      </span>
      {children}
    </label>
  )
}

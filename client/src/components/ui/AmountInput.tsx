import type { ComponentPropsWithRef } from 'react'
import { inputClass } from '../../lib/formStyles'

interface AmountInputProps extends Omit<ComponentPropsWithRef<'input'>, 'type' | 'size'> {
  /** The big, thumb-friendly field the mobile quick-add sheet leads with. */
  large?: boolean
}

export function AmountInput({ className = '', large, ...props }: AmountInputProps) {
  return (
    <div className="relative">
      <input
        type="number"
        inputMode="numeric"
        min={1}
        step={1}
        {...props}
        className={`${inputClass} num [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none ${
          large ? 'h-16 rounded-card pr-12 pl-5 text-[32px] leading-10 font-bold' : 'pr-9 text-right text-[17px] font-bold'
        } ${className}`}
      />
      <span
        aria-hidden="true"
        className={`num pointer-events-none absolute top-1/2 -translate-y-1/2 font-bold text-ink-muted ${large ? 'right-5 text-[26px]' : 'right-3.5 text-[17px]'}`}
      >
        ₫
      </span>
    </div>
  )
}

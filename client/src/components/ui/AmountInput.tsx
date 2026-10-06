import type { InputHTMLAttributes } from 'react'
import { inputClass } from '../../lib/formStyles'

export function AmountInput({ className = '', ...props }: Omit<InputHTMLAttributes<HTMLInputElement>, 'type'>) {
  return (
    <div className="relative">
      <input
        type="number"
        inputMode="numeric"
        min={1}
        step={1}
        {...props}
        className={`${inputClass} tabular pr-9 text-right font-display text-[17px] font-bold [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none ${className}`}
      />
      <span aria-hidden="true" className="pointer-events-none absolute top-1/2 right-3.5 -translate-y-1/2 font-display text-[17px] font-bold text-ink-muted">
        ₫
      </span>
    </div>
  )
}

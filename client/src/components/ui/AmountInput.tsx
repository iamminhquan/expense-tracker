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
        className={`${inputClass} figure-sm pr-9 text-right text-[18px] [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none ${className}`}
      />
      <span aria-hidden="true" className="figure-sm pointer-events-none absolute top-1/2 right-4 -translate-y-1/2 text-[17px] text-ink-muted">
        ₫
      </span>
    </div>
  )
}

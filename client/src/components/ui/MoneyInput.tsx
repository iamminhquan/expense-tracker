import type { InputHTMLAttributes, Ref } from 'react'

const groupFormatter = new Intl.NumberFormat('vi-VN')

interface MoneyInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  /** Whole đồng as digits only, e.g. "45000"; shown grouped as "45.000". */
  digits: string
  onDigitsChange: (digits: string) => void
  type: 'expense' | 'income'
  invalid?: boolean
  ref?: Ref<HTMLInputElement>
}

// The big amount field of the add and edit forms: the sign and ₫ sit beside the typed figure.
export function MoneyInput({ digits, onDigitsChange, type, invalid, ref, ...props }: MoneyInputProps) {
  const expense = type === 'expense'
  return (
    <div
      className={`flex items-center gap-1 rounded-panel border-2 bg-surface-2 px-4 has-focus-visible:border-accent ${invalid ? 'border-danger' : 'border-transparent'}`}
    >
      <span aria-hidden="true" className={`figure text-[40px] leading-none ${expense ? 'text-expense' : 'text-income'}`}>
        {expense ? '−' : '+'}
      </span>
      <input
        {...props}
        ref={ref}
        inputMode="numeric"
        autoComplete="off"
        placeholder="0"
        value={digits ? groupFormatter.format(Number(digits)) : ''}
        onChange={(e) => onDigitsChange(e.target.value.replace(/\D/g, '').replace(/^0+/, '').slice(0, 12))}
        className="figure h-[72px] w-full min-w-0 bg-transparent text-[48px] leading-none text-ink placeholder:text-ink-muted/60 focus-visible:outline-none"
      />
      <span aria-hidden="true" className="figure text-[30px] leading-none text-ink-muted">
        ₫
      </span>
    </div>
  )
}

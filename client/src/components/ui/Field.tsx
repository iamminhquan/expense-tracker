import { useId, type ReactNode } from 'react'
import { labelClass } from '../../lib/formStyles'
import { FieldErrorText } from './FieldErrorText'

export interface FieldControlProps {
  id: string
  'aria-invalid': true | undefined
  'aria-describedby': string | undefined
}

interface FieldProps {
  label: ReactNode
  error?: string | null
  hint?: ReactNode
  className?: string
  children: (control: FieldControlProps) => ReactNode
}

export function Field({ label, error, hint, className, children }: FieldProps) {
  const id = useId()
  const errorId = `${id}-error`
  const hintId = `${id}-hint`
  const describedBy = [error ? errorId : null, hint ? hintId : null].filter(Boolean).join(' ') || undefined

  return (
    <div className={className}>
      <label htmlFor={id} className={labelClass}>
        {label}
      </label>
      {children({ id, 'aria-invalid': error ? true : undefined, 'aria-describedby': describedBy })}
      {hint && !error && (
        <p id={hintId} className="mt-1.5 text-[13px] leading-[18px] text-ink-muted">
          {hint}
        </p>
      )}
      <FieldErrorText id={errorId}>{error}</FieldErrorText>
    </div>
  )
}

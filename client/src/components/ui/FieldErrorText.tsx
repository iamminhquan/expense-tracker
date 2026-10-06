import { CircleAlert } from 'lucide-react'

export function FieldErrorText({ id, children }: { id?: string; children?: string | null }) {
  if (!children) return null
  return (
    <p id={id} className="mt-1.5 flex items-start gap-1.5 text-[13px] leading-[18px] font-medium text-danger">
      <CircleAlert aria-hidden="true" className="mt-px size-4 shrink-0" />
      {children}
    </p>
  )
}

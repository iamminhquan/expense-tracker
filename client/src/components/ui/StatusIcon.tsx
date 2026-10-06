import { CircleCheck, CircleX, Clock, LoaderCircle, Mail } from 'lucide-react'

type StatusKind = 'success' | 'danger' | 'warning' | 'loading' | 'mail'

const kinds: Record<StatusKind, { className: string; Icon: typeof Mail }> = {
  success: { className: 'bg-income-tint text-income', Icon: CircleCheck },
  danger: { className: 'bg-danger-tint text-danger', Icon: CircleX },
  warning: { className: 'bg-warning-tint text-warning', Icon: Clock },
  loading: { className: 'bg-surface-2 text-ink', Icon: LoaderCircle },
  mail: { className: 'bg-surface-2 text-ink', Icon: Mail },
}

export function StatusIcon({ kind }: { kind: StatusKind }) {
  const { className, Icon } = kinds[kind]
  return (
    <span aria-hidden="true" className={`mx-auto flex size-[72px] items-center justify-center rounded-full ${className}`}>
      <Icon className={`size-9 ${kind === 'loading' ? 'animate-spin motion-reduce:animate-none' : ''}`} />
    </span>
  )
}

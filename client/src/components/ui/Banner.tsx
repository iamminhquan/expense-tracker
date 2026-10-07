import type { ReactNode } from 'react'
import { CircleAlert, CircleCheck, Info, TriangleAlert } from 'lucide-react'

type BannerKind = 'info' | 'success' | 'warning' | 'danger'

const styles: Record<BannerKind, { box: string; icon: string; Icon: typeof Info }> = {
  info: { box: 'bg-accent-tint', icon: 'bg-surface text-accent-text', Icon: Info },
  success: { box: 'bg-income-tint', icon: 'bg-surface text-income', Icon: CircleCheck },
  warning: { box: 'bg-warning-tint', icon: 'bg-surface text-warning', Icon: TriangleAlert },
  danger: { box: 'bg-danger-tint', icon: 'bg-surface text-danger', Icon: CircleAlert },
}

interface BannerProps {
  kind?: BannerKind
  title?: ReactNode
  action?: ReactNode
  children?: ReactNode
}

export function Banner({ kind = 'info', title, action, children }: BannerProps) {
  const { box, icon, Icon } = styles[kind]
  return (
    <div role={kind === 'danger' ? 'alert' : undefined} className={`flex flex-wrap items-center gap-x-3 gap-y-3 rounded-card px-4 py-3.5 ${box}`}>
      <span aria-hidden="true" className={`flex size-9 shrink-0 items-center justify-center rounded-full ${icon}`}>
        <Icon className="size-[18px]" />
      </span>
      <div className="min-w-0 flex-1 basis-48 text-[14px] leading-5 font-medium text-ink">
        {title && <p className="font-semibold">{title}</p>}
        {children}
      </div>
      {action}
    </div>
  )
}

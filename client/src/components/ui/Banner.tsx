import type { ReactNode } from 'react'
import { CircleAlert, CircleCheck, Info, TriangleAlert } from 'lucide-react'

type BannerKind = 'info' | 'success' | 'warning' | 'danger'

const styles: Record<BannerKind, { box: string; icon: string; Icon: typeof Info }> = {
  info: { box: 'bg-surface-2', icon: 'text-ink', Icon: Info },
  success: { box: 'bg-income-tint', icon: 'text-income', Icon: CircleCheck },
  warning: { box: 'bg-warning-tint', icon: 'text-warning', Icon: TriangleAlert },
  danger: { box: 'bg-danger-tint', icon: 'text-danger', Icon: CircleAlert },
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
    <div role={kind === 'danger' ? 'alert' : undefined} className={`flex items-start gap-3 rounded-[16px] px-4 py-3.5 ${box}`}>
      <Icon aria-hidden="true" className={`mt-px size-5 shrink-0 ${icon}`} />
      <div className="min-w-0 flex-1 text-[14px] leading-5 font-medium text-ink">
        {title && <p className="font-bold">{title}</p>}
        {children}
      </div>
      {action}
    </div>
  )
}

import { Check, LogOut, Monitor, Smartphone } from 'lucide-react'
import { useRevokeOtherSessions, useRevokeSession } from '../../hooks/useSettings'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { ApiError } from '../../lib/api/client'
import { buttonClass } from '../../lib/formStyles'
import { formatTimestamp } from '../../lib/format'
import { useToast } from '../../lib/toast/ToastContext'
import { Badge } from '../../components/ui/Badge'
import type { Session } from '../../lib/api/types'
import { Card } from './Card'

const MOBILE_DEVICE = /iphone|ipad|android|mobile/i

export function SessionsCard({ sessions }: { sessions: Session[] }) {
  const revokeSession = useRevokeSession()
  const revokeOthers = useRevokeOtherSessions()
  const toast = useToast()
  const isDesktop = useIsDesktop()
  const others = sessions.filter((s) => !s.isCurrent).length

  async function onRevoke(session: Session) {
    try {
      await revokeSession.mutateAsync(session.id)
      toast.success(`Signed out ${session.device}`)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Could not sign that device out.')
    }
  }

  async function onRevokeOthers() {
    try {
      await revokeOthers.mutateAsync()
      toast.success('Signed out everywhere else')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Could not sign the other devices out.')
    }
  }

  return (
    <Card title="Active sessions" description="Devices signed in to your account.">
      <ul className="-my-1 divide-y divide-border">
        {sessions.map((s) => {
          const Icon = MOBILE_DEVICE.test(s.device) ? Smartphone : Monitor
          return (
            <li key={s.id} className="flex items-center gap-3 py-3">
              <span aria-hidden="true" className="flex size-11 shrink-0 items-center justify-center rounded-[14px] bg-surface-2 text-ink">
                <Icon className="size-5" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{s.device}</p>
                <p className="tabular text-[13px] leading-[18px] text-ink-muted">Signed in {formatTimestamp(s.createdAt)}</p>
              </div>
              {s.isCurrent ? (
                <Badge kind="income" icon={<Check aria-hidden="true" />}>
                  This device
                </Badge>
              ) : (
                <button
                  type="button"
                  onClick={() => void onRevoke(s)}
                  disabled={revokeSession.isPending}
                  aria-label={`Revoke ${s.device}, signed in ${formatTimestamp(s.createdAt)}`}
                  className={buttonClass('secondary', isDesktop ? 'sm' : 'md')}
                >
                  Revoke
                </button>
              )}
            </li>
          )
        })}
      </ul>
      {others > 0 && (
        <button type="button" onClick={() => void onRevokeOthers()} disabled={revokeOthers.isPending} className={`${buttonClass('secondary')} max-sm:w-full`}>
          <LogOut aria-hidden="true" />
          Log out everywhere else
        </button>
      )}
    </Card>
  )
}

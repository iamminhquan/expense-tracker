import { useRevokeOtherSessions, useRevokeSession } from '../../hooks/useSettings'
import { formatTimestamp } from '../../lib/format'
import type { Session } from '../../lib/api/types'
import { Card } from './Card'

interface SessionsCardProps {
  sessions: Session[]
}

export function SessionsCard({ sessions }: SessionsCardProps) {
  const revokeSession = useRevokeSession()
  const revokeOthers = useRevokeOtherSessions()

  return (
    <Card title="Active sessions">
      <ul className="mb-3 space-y-2">
        {sessions.map((s) => (
          <li key={s.id} className="flex items-center justify-between text-[13px]">
            <span className="text-ink-muted">
              {s.device} {s.isCurrent && <span className="text-accent">(this device)</span>}
              <br />
              <span className="text-[12px] text-ink-faint">{formatTimestamp(s.createdAt)}</span>
            </span>
            {!s.isCurrent && (
              <button onClick={() => void revokeSession.mutate(s.id)} className="text-ink-faint hover:text-expense">
                Sign out
              </button>
            )}
          </li>
        ))}
      </ul>
      <button onClick={() => void revokeOthers.mutate()} className="text-[13px] text-accent hover:underline">
        Log out everywhere else
      </button>
    </Card>
  )
}

import { useSettings } from '../../hooks/useSettings'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { pageTitleClass } from '../../lib/formStyles'
import { DangerZoneCard } from './DangerZoneCard'
import { EmailCard } from './EmailCard'
import { PasswordCard } from './PasswordCard'
import { ProfileCard } from './ProfileCard'
import { SessionsCard } from './SessionsCard'

export function SettingsPage() {
  const { data, error, refetch } = useSettings()
  if (!data) {
    if (error) return <InlineError message="Could not load your settings." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading your settings…" />
  }

  return (
    <div className="space-y-4 md:space-y-5">
      <h1 className={pageTitleClass}>Settings</h1>
      <div className="flex items-center gap-4 rounded-card bg-accent-tint p-4 md:p-5">
        <span aria-hidden="true" className="flex size-14 shrink-0 items-center justify-center rounded-full bg-accent font-display text-[26px] font-bold text-on-accent">
          {(data.name || data.username)[0]?.toUpperCase()}
        </span>
        <div className="min-w-0">
          <p className="truncate font-display text-[20px] leading-6 font-semibold tracking-[-0.015em] text-ink">{data.name}</p>
          <p className="truncate text-[14px] leading-5 text-ink-muted">
            @{data.username} · {data.email}
          </p>
        </div>
      </div>
      <div className="grid items-start gap-4 md:gap-5 lg:grid-cols-2">
        <div className="space-y-4 md:space-y-5">
          <ProfileCard name={data.name} username={data.username} />
          <PasswordCard />
        </div>
        <div className="space-y-4 md:space-y-5">
          <EmailCard email={data.email} pendingEmail={data.pendingEmail} />
          <SessionsCard sessions={data.sessions} />
        </div>
        <DangerZoneCard />
      </div>
    </div>
  )
}

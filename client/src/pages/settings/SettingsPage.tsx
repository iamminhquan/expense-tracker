import { useSettings } from '../../hooks/useSettings'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { pageTitleClass } from '../../lib/formStyles'
import { AppearanceCard } from './AppearanceCard'
import { DangerZoneCard } from './DangerZoneCard'
import { EmailCard } from './EmailCard'
import { PasswordCard } from './PasswordCard'
import { ProfileCard } from './ProfileCard'
import { SessionsCard } from './SessionsCard'

export function SettingsPage() {
  const { data, error, refetch } = useSettings()
  if (!data) {
    if (error) return <InlineError message="Could not load your settings." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading your settings…" shape="cards" />
  }

  return (
    <div className="space-y-5 md:space-y-7">
      <div className="flex items-center gap-4">
        <span aria-hidden="true" className="heading flex size-14 shrink-0 items-center justify-center rounded-[20px] bg-accent text-[24px] text-on-accent md:size-16">
          {(data.name || data.username || '?')[0].toUpperCase()}
        </span>
        <div className="min-w-0">
          <h1 className={pageTitleClass}>Settings</h1>
          <p className="truncate text-[14px] leading-5 text-ink-muted">
            {data.name}, @{data.username}
          </p>
        </div>
      </div>
      <div className="grid items-start gap-4 md:gap-5 lg:grid-cols-2">
        <div className="space-y-4 md:space-y-5">
          <ProfileCard name={data.name} username={data.username} />
          <AppearanceCard />
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

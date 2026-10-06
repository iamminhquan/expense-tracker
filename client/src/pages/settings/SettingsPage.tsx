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
    <div className="space-y-4 md:space-y-6">
      <h1 className={pageTitleClass}>Settings</h1>
      <div className="grid items-start gap-4 md:gap-6 lg:grid-cols-2">
        <div className="space-y-4 md:space-y-6">
          <ProfileCard name={data.name} username={data.username} />
          <PasswordCard />
        </div>
        <div className="space-y-4 md:space-y-6">
          <EmailCard email={data.email} pendingEmail={data.pendingEmail} />
          <SessionsCard sessions={data.sessions} />
        </div>
        <DangerZoneCard />
      </div>
    </div>
  )
}

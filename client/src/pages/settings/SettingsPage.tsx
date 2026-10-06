import { useSettings } from '../../hooks/useSettings'
import { DangerZoneCard } from './DangerZoneCard'
import { EmailCard } from './EmailCard'
import { PasswordCard } from './PasswordCard'
import { ProfileCard } from './ProfileCard'
import { SessionsCard } from './SessionsCard'

export function SettingsPage() {
  const { data, isLoading } = useSettings()
  if (isLoading || !data) return <p className="text-ink-faint">Loading…</p>

  return (
    <div className="max-w-[560px] space-y-6">
      <h1 className="text-[20px] font-semibold">Settings</h1>
      <ProfileCard name={data.name} username={data.username} />
      <EmailCard email={data.email} pendingEmail={data.pendingEmail} />
      <PasswordCard />
      <SessionsCard sessions={data.sessions} />
      <DangerZoneCard />
    </div>
  )
}

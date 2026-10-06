import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  useDeleteAccount,
  useResendVerification,
  useRevokeOtherSessions,
  useRevokeSession,
  useSettings,
  useUpdateEmail,
  useUpdatePassword,
  useUpdateProfile,
} from '../hooks/useSettings'
import { useAuth } from '../lib/auth/AuthContext'
import { ApiError } from '../lib/api/client'
import { formatTimestamp } from '../lib/format'
import { inputClass, primaryButtonClass } from '../lib/formStyles'

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

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-[16px] border border-border-card bg-surface p-5">
      <p className="mb-3 text-[14px] font-semibold">{title}</p>
      {children}
    </div>
  )
}

function ProfileCard({ name, username }: { name: string; username: string }) {
  const updateProfile = useUpdateProfile()
  const [form, setForm] = useState({ name, username })
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaved(false)
    try {
      await updateProfile.mutateAsync(form)
      setSaved(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update profile.')
    }
  }

  return (
    <Card title="Profile">
      <form onSubmit={onSubmit} className="space-y-3">
        <input
          className={inputClass}
          value={form.name}
          onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
          placeholder="Name"
        />
        <input
          className={inputClass}
          value={form.username}
          onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
          placeholder="Username"
        />
        {error && <p className="text-[13px] text-expense">{error}</p>}
        {saved && <p className="text-[13px] text-income">Profile updated.</p>}
        <button type="submit" disabled={updateProfile.isPending} className={primaryButtonClass}>
          Save
        </button>
      </form>
    </Card>
  )
}

function EmailCard({ email, pendingEmail }: { email: string; pendingEmail?: string }) {
  const updateEmail = useUpdateEmail()
  const resendVerification = useResendVerification()
  const [newEmail, setNewEmail] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateEmail.mutateAsync({ email: newEmail, currentPassword })
      setSaved(true)
      setNewEmail('')
      setCurrentPassword('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update email.')
    }
  }

  return (
    <Card title="Email">
      <p className="mb-3 text-[13px] text-ink-muted">
        Current: <span className="text-ink">{email}</span>
      </p>
      {pendingEmail && (
        <p className="mb-3 text-[13px] text-ink-faint">
          Pending confirmation: {pendingEmail}{' '}
          <button onClick={() => void resendVerification.mutate()} className="text-accent hover:underline">
            Resend link
          </button>
        </p>
      )}
      <form onSubmit={onSubmit} className="space-y-3">
        <input
          type="email"
          className={inputClass}
          value={newEmail}
          onChange={(e) => setNewEmail(e.target.value)}
          placeholder="New email"
          required
        />
        <input
          type="password"
          className={inputClass}
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          placeholder="Current password"
          required
        />
        {error && <p className="text-[13px] text-expense">{error}</p>}
        {saved && <p className="text-[13px] text-income">Check your inbox to confirm the new address.</p>}
        <button type="submit" disabled={updateEmail.isPending} className={primaryButtonClass}>
          Change email
        </button>
      </form>
    </Card>
  )
}

function PasswordCard() {
  const updatePassword = useUpdatePassword()
  const [form, setForm] = useState({ currentPassword: '', newPassword: '', newPasswordConfirm: '' })
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updatePassword.mutateAsync(form)
      setSaved(true)
      setForm({ currentPassword: '', newPassword: '', newPasswordConfirm: '' })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update password.')
    }
  }

  return (
    <Card title="Password">
      <form onSubmit={onSubmit} className="space-y-3">
        <input
          type="password"
          className={inputClass}
          value={form.currentPassword}
          onChange={(e) => setForm((f) => ({ ...f, currentPassword: e.target.value }))}
          placeholder="Current password"
          required
        />
        <input
          type="password"
          className={inputClass}
          value={form.newPassword}
          onChange={(e) => setForm((f) => ({ ...f, newPassword: e.target.value }))}
          placeholder="New password"
          required
        />
        <input
          type="password"
          className={inputClass}
          value={form.newPasswordConfirm}
          onChange={(e) => setForm((f) => ({ ...f, newPasswordConfirm: e.target.value }))}
          placeholder="Confirm new password"
          required
        />
        {error && <p className="text-[13px] text-expense">{error}</p>}
        {saved && <p className="text-[13px] text-income">Password updated.</p>}
        <button type="submit" disabled={updatePassword.isPending} className={primaryButtonClass}>
          Update password
        </button>
      </form>
    </Card>
  )
}

function SessionsCard({ sessions }: { sessions: { id: string; device: string; createdAt: string; isCurrent: boolean }[] }) {
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

function DangerZoneCard() {
  const deleteAccount = useDeleteAccount()
  const { logout } = useAuth()
  const navigate = useNavigate()
  const [currentPassword, setCurrentPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onDelete(e: React.FormEvent) {
    e.preventDefault()
    if (!confirm('Delete your account? This cannot be undone.')) return
    setError(null)
    try {
      await deleteAccount.mutateAsync(currentPassword)
      await logout()
      navigate('/login', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete account.')
    }
  }

  return (
    <div className="rounded-[16px] border border-danger-border bg-danger-tint p-5">
      <p className="mb-3 text-[14px] font-semibold text-expense">Danger zone</p>
      <form onSubmit={onDelete} className="space-y-3">
        <input
          type="password"
          className={inputClass}
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          placeholder="Current password"
          required
        />
        {error && <p className="text-[13px] text-expense">{error}</p>}
        <button
          type="submit"
          disabled={deleteAccount.isPending}
          className="w-full rounded-[10px] bg-expense px-4 py-2.5 text-[14px] font-semibold text-on-solid hover:opacity-90 disabled:opacity-50"
        >
          Delete account
        </button>
      </form>
    </div>
  )
}

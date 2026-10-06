import { useState, type FormEvent } from 'react'
import { useResendVerification, useUpdateEmail } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { inputClass, primaryButtonClass } from '../../lib/formStyles'
import { Card } from './Card'

interface EmailCardProps {
  email: string
  pendingEmail?: string
}

export function EmailCard({ email, pendingEmail }: EmailCardProps) {
  const updateEmail = useUpdateEmail()
  const resendVerification = useResendVerification()
  const [newEmail, setNewEmail] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  async function onSubmit(e: FormEvent) {
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

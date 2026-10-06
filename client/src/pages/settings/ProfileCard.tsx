import { useState, type FormEvent } from 'react'
import { useUpdateProfile } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { inputClass, primaryButtonClass } from '../../lib/formStyles'
import { Card } from './Card'

interface ProfileCardProps {
  name: string
  username: string
}

export function ProfileCard({ name, username }: ProfileCardProps) {
  const updateProfile = useUpdateProfile()
  const [form, setForm] = useState({ name, username })
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
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

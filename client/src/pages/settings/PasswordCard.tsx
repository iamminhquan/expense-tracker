import { useState, type FormEvent } from 'react'
import { useUpdatePassword } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { inputClass, primaryButtonClass } from '../../lib/formStyles'
import { Card } from './Card'

export function PasswordCard() {
  const updatePassword = useUpdatePassword()
  const [form, setForm] = useState({ currentPassword: '', newPassword: '', newPasswordConfirm: '' })
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  async function onSubmit(e: FormEvent) {
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

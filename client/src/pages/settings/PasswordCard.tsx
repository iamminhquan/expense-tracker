import { useState, type FormEvent } from 'react'
import { useUpdatePassword } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { buttonClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { Card } from './Card'

const EMPTY = { currentPassword: '', newPassword: '', newPasswordConfirm: '' }

export function PasswordCard() {
  const updatePassword = useUpdatePassword()
  const toast = useToast()
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updatePassword.mutateAsync(form)
      setForm(EMPTY)
      toast.success('Password updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update password.')
    }
  }

  return (
    <Card title="Password" description="Changing it signs out every other device.">
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="Current password">
          {(control) => (
            <PasswordInput {...control} required autoComplete="current-password" value={form.currentPassword} onChange={(e) => setForm((f) => ({ ...f, currentPassword: e.target.value }))} />
          )}
        </Field>
        <Field label="New password">
          {(control) => (
            <PasswordInput {...control} required autoComplete="new-password" value={form.newPassword} onChange={(e) => setForm((f) => ({ ...f, newPassword: e.target.value }))} />
          )}
        </Field>
        <Field label="Confirm new password">
          {(control) => (
            <PasswordInput
              {...control}
              required
              autoComplete="new-password"
              value={form.newPasswordConfirm}
              onChange={(e) => setForm((f) => ({ ...f, newPasswordConfirm: e.target.value }))}
            />
          )}
        </Field>
        {error && (
          <div role="alert">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
        <button type="submit" disabled={updatePassword.isPending} aria-busy={updatePassword.isPending} className={`${buttonClass('primary')} max-sm:h-[52px] max-sm:w-full`}>
          Update password
        </button>
      </form>
    </Card>
  )
}

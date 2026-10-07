import { useState, type FormEvent } from 'react'
import { useUpdateProfile } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { Card } from './Card'

interface ProfileCardProps {
  name: string
  username: string
}

export function ProfileCard({ name, username }: ProfileCardProps) {
  const updateProfile = useUpdateProfile()
  const toast = useToast()
  const [form, setForm] = useState({ name, username })
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateProfile.mutateAsync(form)
      toast.success('Profile saved')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update profile.')
    }
  }

  return (
    <Card title="Profile" description="How you appear in $pend.">
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="Name">
          {(control) => <input {...control} required className={inputClass} value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />}
        </Field>
        <Field label="Username" hint="3–20 characters: lowercase letters, numbers or underscores, starting with a letter.">
          {(control) => (
            <input
              {...control}
              required
              autoCapitalize="none"
              className={inputClass}
              value={form.username}
              onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
            />
          )}
        </Field>
        {error && (
          <div role="alert">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
        <button type="submit" disabled={updateProfile.isPending} aria-busy={updateProfile.isPending} className={`${buttonClass('primary')} max-sm:h-[52px] max-sm:w-full`}>
          Save profile
        </button>
      </form>
    </Card>
  )
}

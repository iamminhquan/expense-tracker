import { useState, type FormEvent } from 'react'
import { Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useDeleteAccount } from '../../hooks/useSettings'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'
import { buttonClass } from '../../lib/formStyles'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { Field } from '../../components/ui/Field'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { Card } from './Card'

export function DangerZoneCard() {
  const deleteAccount = useDeleteAccount()
  const { logout } = useAuth()
  const navigate = useNavigate()
  const [currentPassword, setCurrentPassword] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setConfirmOpen(true)
  }

  async function onDelete() {
    try {
      await deleteAccount.mutateAsync(currentPassword)
      await logout()
      navigate('/login', { replace: true })
    } catch (err) {
      setConfirmOpen(false)
      setError(err instanceof ApiError ? err.message : 'Could not delete account.')
    }
  }

  return (
    <Card
      title="Danger zone"
      danger
      className="lg:col-span-2"
      description="Deleting your account removes every transaction and every category you made. It happens immediately and can't be undone. Export a copy from Your data first."
    >
      <form onSubmit={onSubmit} className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start">
        <Field label="Current password" error={error}>
          {(control) => <PasswordInput {...control} required autoComplete="current-password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />}
        </Field>
        <button type="submit" className={`${buttonClass('danger')} sm:mt-6`}>
          <Trash2 aria-hidden="true" />
          Delete account
        </button>
      </form>
      <ConfirmDialog
        open={confirmOpen}
        title="Delete your account?"
        confirmLabel="Delete account"
        onCancel={() => setConfirmOpen(false)}
        onConfirm={() => void onDelete()}
        pending={deleteAccount.isPending}
      >
        Every transaction and category you made will be gone for good. You can sign up again with the same email afterwards, but nothing comes back.
      </ConfirmDialog>
    </Card>
  )
}

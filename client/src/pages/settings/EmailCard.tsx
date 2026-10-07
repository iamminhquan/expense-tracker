import { useState, type FormEvent } from 'react'
import { useResendVerification, useUpdateEmail } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Banner } from '../../components/ui/Banner'
import { Field } from '../../components/ui/Field'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { Card } from './Card'

interface EmailCardProps {
  email: string
  pendingEmail?: string
}

export function EmailCard({ email, pendingEmail }: EmailCardProps) {
  const updateEmail = useUpdateEmail()
  const resendVerification = useResendVerification()
  const toast = useToast()
  const [newEmail, setNewEmail] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await updateEmail.mutateAsync({ email: newEmail, currentPassword })
      setNewEmail('')
      setCurrentPassword('')
      toast.success('Check your inbox to confirm the new address')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update email.')
    }
  }

  async function onResend() {
    try {
      await resendVerification.mutateAsync()
      toast.success('Verification link sent')
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : 'Could not send the link.')
    }
  }

  return (
    <Card
      title="Email"
      description={
        <>
          You sign in with <span className="font-semibold text-ink">{email}</span>.
        </>
      }
    >
      {pendingEmail && (
        <Banner
          kind="warning"
          title="Waiting for verification"
          action={
            <button type="button" onClick={() => void onResend()} disabled={resendVerification.isPending} className={buttonClass('secondary', 'sm')}>
              Resend
            </button>
          }
        >
          We sent a link to <span className="font-semibold break-all">{pendingEmail}</span>. Your email stays the same until you open it.
        </Banner>
      )}
      <form onSubmit={onSubmit} className="space-y-4">
        <Field label="New email">
          {(control) => <input {...control} type="email" required autoComplete="email" className={inputClass} value={newEmail} onChange={(e) => setNewEmail(e.target.value)} />}
        </Field>
        <Field label="Current password">
          {(control) => <PasswordInput {...control} required autoComplete="current-password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />}
        </Field>
        {error && (
          <div role="alert">
            <FieldErrorText>{error}</FieldErrorText>
          </div>
        )}
        <button type="submit" disabled={updateEmail.isPending} aria-busy={updateEmail.isPending} className={`${buttonClass('primary')} max-sm:h-[52px] max-sm:w-full`}>
          Change email
        </button>
      </form>
    </Card>
  )
}

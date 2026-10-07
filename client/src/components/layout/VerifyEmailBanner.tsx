import { useState } from 'react'
import { useResendVerification } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { useAuth } from '../../lib/auth/AuthContext'
import { buttonClass } from '../../lib/formStyles'
import { Banner } from '../ui/Banner'

type Resend = 'idle' | 'sent' | { error: string }

// A reminder only: an unconfirmed account can still use everything.
export function VerifyEmailBanner() {
  const { user } = useAuth()
  const resendVerification = useResendVerification()
  const [resend, setResend] = useState<Resend>('idle')

  if (!user || user.emailVerified) return null

  async function onResend() {
    setResend('idle')
    try {
      await resendVerification.mutateAsync()
      setResend('sent')
    } catch (err) {
      setResend({ error: err instanceof ApiError ? err.message : 'Could not send the link.' })
    }
  }

  return (
    <section aria-label="Email confirmation" className="mb-6">
      <Banner
        kind="warning"
        title="Please confirm your email address"
        action={
          <button
            type="button"
            onClick={() => void onResend()}
            disabled={resendVerification.isPending}
            aria-busy={resendVerification.isPending}
            className={buttonClass('secondary', 'sm')}
          >
            Resend link
          </button>
        }
      >
        We sent a link to <span className="font-semibold [overflow-wrap:anywhere]">{user.email}</span>.
        <p role="status" className={`mt-1 empty:hidden ${typeof resend === 'object' ? 'text-danger' : 'text-income'}`}>
          {resend === 'sent' ? 'Link sent. Check your inbox.' : typeof resend === 'object' ? resend.error : ''}
        </p>
      </Banner>
    </section>
  )
}

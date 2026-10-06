import { useState } from 'react'
import { useResendVerification } from '../../hooks/useSettings'
import { ApiError } from '../../lib/api/client'
import { useAuth } from '../../lib/auth/AuthContext'

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
    <section aria-label="Email confirmation" className="border-b border-border-nav bg-track">
      <div className="mx-auto flex w-full max-w-[1280px] flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2 text-[13px] text-ink-muted md:px-9">
        <p>
          Please confirm your email address. We sent a link to <span className="text-ink">{user.email}</span>.
        </p>
        <button
          type="button"
          onClick={() => void onResend()}
          disabled={resendVerification.isPending}
          className="text-accent hover:underline disabled:opacity-60"
        >
          {resendVerification.isPending ? 'Sending…' : 'Resend link'}
        </button>
        <p role="status" className={resend !== 'idle' && typeof resend === 'object' ? 'text-expense' : 'text-income'}>
          {resend === 'sent' ? 'Link sent. Check your inbox.' : typeof resend === 'object' ? resend.error : ''}
        </p>
      </div>
    </section>
  )
}

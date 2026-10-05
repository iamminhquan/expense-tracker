import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { verifyEmail } from '../../lib/api/auth'

type Outcome = 'checking' | 'verified' | 'conflict' | 'invalid'

/** Mirrors verify_email.html's three outcomes. Public: the browser that opens this link is often not the one the visitor is signed in on. */
export function VerifyEmailPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [outcome, setOutcome] = useState<Outcome>('checking')

  useEffect(() => {
    if (!token) {
      setOutcome('invalid')
      return
    }
    verifyEmail(token)
      .then((res) => setOutcome(res.verified ? 'verified' : res.conflict ? 'conflict' : 'invalid'))
      .catch(() => setOutcome('invalid'))
  }, [token])

  return (
    <AuthLayout>
      {outcome === 'checking' && <p className="text-center text-[14px] text-ink-faint">Verifying…</p>}
      {outcome === 'verified' && <p className="text-center text-[14px] text-income">Your email is verified.</p>}
      {outcome === 'conflict' && (
        <p className="text-center text-[14px] text-ink-muted">
          That address is already registered to another account.
        </p>
      )}
      {outcome === 'invalid' && (
        <p className="text-center text-[14px] text-ink-muted">That verification link is invalid or has expired.</p>
      )}
      <p className="mt-4 text-center text-[13px] text-ink-faint">
        <Link to="/dashboard" className="text-accent hover:underline">
          Go to dashboard
        </Link>
      </p>
    </AuthLayout>
  )
}

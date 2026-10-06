import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { verifyEmail } from '../../lib/api/auth'
import { useAuth } from '../../lib/auth/AuthContext'

type Outcome = 'checking' | 'verified' | 'conflict' | 'invalid'

// Public route: the emailed link is often opened in a browser that isn't signed in.
export function VerifyEmailPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [outcome, setOutcome] = useState<Outcome>('checking')
  const { reloadUser } = useAuth()

  useEffect(() => {
    if (!token) {
      setOutcome('invalid')
      return
    }
    verifyEmail(token)
      .then((res) => {
        setOutcome(res.verified ? 'verified' : res.conflict ? 'conflict' : 'invalid')
        if (res.verified) void reloadUser()
      })
      .catch(() => setOutcome('invalid'))
  }, [token, reloadUser])

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

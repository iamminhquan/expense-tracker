import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { StatusIcon } from '../../components/ui/StatusIcon'
import { authTitleClass, buttonClass } from '../../lib/formStyles'
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
      <div aria-live="polite" className="space-y-3 text-center">
        {outcome === 'checking' && (
          <>
            <StatusIcon kind="loading" />
            <p className="text-[15px] text-ink-muted">Verifying your email…</p>
          </>
        )}
        {outcome === 'verified' && (
          <>
            <StatusIcon kind="success" />
            <h1 className={authTitleClass}>Your email is verified</h1>
            <p className="text-[15px] leading-[22px] text-ink-muted">You're all set.</p>
          </>
        )}
        {outcome === 'conflict' && (
          <>
            <StatusIcon kind="danger" />
            <h1 className={authTitleClass}>That address is taken</h1>
            <p className="text-[15px] leading-[22px] text-ink-muted">It's already registered to another account, so your email wasn't changed.</p>
          </>
        )}
        {outcome === 'invalid' && (
          <>
            <StatusIcon kind="danger" />
            <h1 className={authTitleClass}>This link didn't work</h1>
            <p className="text-[15px] leading-[22px] text-ink-muted">
              It's invalid or has expired. Log in and request a new link from Settings.
            </p>
          </>
        )}
      </div>
      {outcome !== 'checking' && (
        <Link to={outcome === 'invalid' ? '/login' : '/dashboard'} className={`${buttonClass('primary')} w-full`}>
          {outcome === 'invalid' ? 'Log in' : 'Go to Overview'}
        </Link>
      )}
    </AuthLayout>
  )
}

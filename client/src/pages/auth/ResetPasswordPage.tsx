import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { Banner } from '../../components/ui/Banner'
import { Field } from '../../components/ui/Field'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { StatusIcon } from '../../components/ui/StatusIcon'
import { authTitleClass, buttonClass } from '../../lib/formStyles'
import { checkResetToken, resetPassword } from '../../lib/api/auth'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'


export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const { setSession } = useAuth()

  const [checking, setChecking] = useState(true)
  const [invalid, setInvalid] = useState(false)
  const [password, setPassword] = useState('')
  const [passwordConfirm, setPasswordConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!token) {
      setInvalid(true)
      setChecking(false)
      return
    }
    checkResetToken(token)
      .then(() => setChecking(false))
      .catch(() => {
        setInvalid(true)
        setChecking(false)
      })
  }, [token])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const res = await resetPassword(token, password, passwordConfirm)
      // A successful reset signs the visitor in, like login.
      setSession(res)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong, please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  if (checking) {
    return (
      <AuthLayout>
        <div role="status" className="space-y-3 text-center">
          <StatusIcon kind="loading" />
          <p className="text-[15px] text-ink-muted">Checking your link…</p>
        </div>
      </AuthLayout>
    )
  }

  if (invalid) {
    return (
      <AuthLayout>
        <div className="space-y-3 text-center">
          <StatusIcon kind="warning" />
          <h1 className={authTitleClass}>This link has expired</h1>
          <p className="text-[15px] leading-[22px] text-ink-muted">Reset links work once, for one hour. Ask for a new one and use it straight away.</p>
        </div>
        <Link to="/forgot-password" className={`${buttonClass('primary')} w-full`}>
          Request a new link
        </Link>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout>
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <h1 className={authTitleClass}>Choose a new password</h1>
          <p className="mt-1.5 text-[15px] leading-[22px] text-ink-muted">You'll be signed in as soon as it's saved.</p>
        </div>
        {error && <Banner kind="danger">{error}</Banner>}
        <Field label="New password">
          {(control) => <PasswordInput {...control} required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />}
        </Field>
        <Field label="Confirm new password">
          {(control) => (
            <PasswordInput {...control} required autoComplete="new-password" value={passwordConfirm} onChange={(e) => setPasswordConfirm(e.target.value)} />
          )}
        </Field>
        <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary')} w-full`}>
          {submitting ? 'Saving…' : 'Save and sign in'}
        </button>
      </form>
    </AuthLayout>
  )
}

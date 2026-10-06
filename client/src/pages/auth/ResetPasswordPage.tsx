import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { FieldError } from '../../components/FieldError'
import { inputClass, primaryButtonClass } from '../../lib/formStyles'
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
        <p className="text-center text-[14px] text-ink-faint">Checking your link…</p>
      </AuthLayout>
    )
  }

  if (invalid) {
    return (
      <AuthLayout>
        <p className="text-center text-[14px] text-ink-muted">
          That reset link is invalid or has expired.{' '}
          <Link to="/forgot-password" className="text-accent hover:underline">
            Request a new one
          </Link>
          .
        </p>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout>
      <form onSubmit={onSubmit} className="space-y-4">
        <p className="text-[14px] text-ink-muted">Choose a new password for your account.</p>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="password">
            New password
          </label>
          <input
            id="password"
            type="password"
            required
            className={inputClass}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="password-confirm">
            Confirm new password
          </label>
          <input
            id="password-confirm"
            type="password"
            required
            className={inputClass}
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
          />
        </div>
        <FieldError>{error}</FieldError>
        <button type="submit" disabled={submitting} className={primaryButtonClass}>
          {submitting ? 'Saving…' : 'Reset password'}
        </button>
      </form>
    </AuthLayout>
  )
}

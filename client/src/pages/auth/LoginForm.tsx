import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Banner } from '../../components/ui/Banner'
import { Field } from '../../components/ui/Field'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'

export function LoginForm() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await login(email, password)
      const from = (location.state as { from?: Location })?.from?.pathname ?? '/dashboard'
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong, please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      {error && <Banner kind="danger">{error}</Banner>}
      <Field label="Email">
        {(control) => <input {...control} type="email" required autoComplete="email" className={inputClass} value={email} onChange={(e) => setEmail(e.target.value)} />}
      </Field>
      <Field label="Password">
        {(control) => <PasswordInput {...control} required autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />}
      </Field>
      <div className="flex justify-end">
        <Link to="/forgot-password" className="inline-flex min-h-11 items-center text-[14px] font-semibold text-accent-text underline underline-offset-2">
          Forgot password?
        </Link>
      </div>
      <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary')} w-full`}>
        {submitting ? 'Logging in…' : 'Log in'}
      </button>
    </form>
  )
}

import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { AuthLayout, FieldError, inputClass, primaryButtonClass } from '../../components/layout/AuthLayout'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'

export function AuthPage({ tab }: { tab: 'login' | 'register' }) {
  return <AuthLayout>{tab === 'login' ? <LoginForm /> : <RegisterForm />}</AuthLayout>
}

function Tabs({ active }: { active: 'login' | 'register' }) {
  return (
    <div className="mb-6 flex gap-1 rounded-[10px] bg-track p-[3px]">
      <Link
        to="/login"
        className={`flex-1 rounded-[8px] py-2 text-center text-[13px] ${
          active === 'login' ? 'bg-surface font-semibold text-ink shadow-sm' : 'text-ink-faint'
        }`}
      >
        Log in
      </Link>
      <Link
        to="/register"
        className={`flex-1 rounded-[8px] py-2 text-center text-[13px] ${
          active === 'register' ? 'bg-surface font-semibold text-ink shadow-sm' : 'text-ink-faint'
        }`}
      >
        Sign up
      </Link>
    </div>
  )
}

function LoginForm() {
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
    <>
      <Tabs active="login" />
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="email">
            Email
          </label>
          <input id="email" type="email" required className={inputClass} value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="password">
            Password
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
        <FieldError>{error}</FieldError>
        <button type="submit" disabled={submitting} className={primaryButtonClass}>
          {submitting ? 'Logging in…' : 'Log in'}
        </button>
        <p className="text-center text-[13px] text-ink-faint">
          <Link to="/forgot-password" className="text-accent hover:underline">
            Forgot password?
          </Link>
        </p>
      </form>
    </>
  )
}

function RegisterForm() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ name: '', email: '', username: '', password: '', passwordConfirm: '' })
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [field]: e.target.value }))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await register(form)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong, please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Tabs active="register" />
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="name">
            Name
          </label>
          <input id="name" required className={inputClass} value={form.name} onChange={update('name')} />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="reg-email">
            Email
          </label>
          <input id="reg-email" type="email" required className={inputClass} value={form.email} onChange={update('email')} />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="username">
            Username
          </label>
          <input id="username" required className={inputClass} value={form.username} onChange={update('username')} />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="reg-password">
            Password
          </label>
          <input
            id="reg-password"
            type="password"
            required
            className={inputClass}
            value={form.password}
            onChange={update('password')}
          />
        </div>
        <div>
          <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="password-confirm">
            Confirm password
          </label>
          <input
            id="password-confirm"
            type="password"
            required
            className={inputClass}
            value={form.passwordConfirm}
            onChange={update('passwordConfirm')}
          />
        </div>
        <FieldError>{error}</FieldError>
        <button type="submit" disabled={submitting} className={primaryButtonClass}>
          {submitting ? 'Creating account…' : 'Create account'}
        </button>
      </form>
    </>
  )
}

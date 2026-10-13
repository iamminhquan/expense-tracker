import { useState, type ChangeEvent, type FormEvent, type ReactNode } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Banner } from '../../components/ui/Banner'
import { Field } from '../../components/ui/Field'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'

type Mode = 'login' | 'register'

function Row({ open = true, children }: { open?: boolean; children: ReactNode }) {
  return (
    <div className="reveal" data-open={open} inert={!open}>
      <div>
        <div className="pb-3">{children}</div>
      </div>
    </div>
  )
}

/* One form for both tabs, so switching keeps what was typed and the extra sign-up rows slide open. */
export function AuthForm({ mode }: { mode: Mode }) {
  const { login, register } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [form, setForm] = useState({ name: '', email: '', username: '', password: '', passwordConfirm: '' })
  const [error, setError] = useState<{ mode: Mode; message: string } | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const isRegister = mode === 'register'

  function update(field: keyof typeof form) {
    return (e: ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [field]: e.target.value }))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      if (isRegister) {
        await register(form)
        navigate('/dashboard', { replace: true })
      } else {
        await login(form.email, form.password)
        const from = (location.state as { from?: Location })?.from?.pathname ?? '/dashboard'
        navigate(from, { replace: true })
      }
    } catch (err) {
      setError({ mode, message: err instanceof ApiError ? err.message : 'Something went wrong, please try again.' })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={onSubmit}>
      {error?.mode === mode && (
        <div className="pb-3">
          <Banner kind="danger">{error.message}</Banner>
        </div>
      )}
      <Row open={isRegister}>
        <Field label="Name">
          {(control) => <input {...control} required={isRegister} autoComplete="name" className={inputClass} value={form.name} onChange={update('name')} />}
        </Field>
      </Row>
      <Row>
        <Field label="Email">
          {(control) => <input {...control} type="email" required autoComplete="email" className={inputClass} value={form.email} onChange={update('email')} />}
        </Field>
      </Row>
      <Row open={isRegister}>
        <Field
          label={
            <span className="flex flex-wrap items-baseline justify-between gap-x-3">
              Username
              <span className="text-[12px] font-normal text-ink-muted">3–20 characters, a–z, 0–9 or _, letter first</span>
            </span>
          }
        >
          {(control) => (
            <input {...control} required={isRegister} autoCapitalize="none" autoComplete="username" className={inputClass} value={form.username} onChange={update('username')} />
          )}
        </Field>
      </Row>
      <Row>
        <Field label="Password">
          {(control) => (
            <PasswordInput
              {...control}
              required
              autoComplete={isRegister ? 'new-password' : 'current-password'}
              value={form.password}
              onChange={update('password')}
            />
          )}
        </Field>
      </Row>
      <Row open={isRegister}>
        <Field label="Confirm password">
          {(control) => <PasswordInput {...control} required={isRegister} autoComplete="new-password" value={form.passwordConfirm} onChange={update('passwordConfirm')} />}
        </Field>
      </Row>
      <Row open={!isRegister}>
        <div className="flex justify-end">
          <Link to="/forgot-password" className="inline-flex min-h-11 items-center text-[14px] font-semibold text-accent-text underline underline-offset-2">
            Forgot password?
          </Link>
        </div>
      </Row>
      <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary')} w-full`}>
        {isRegister ? (submitting ? 'Creating account…' : 'Create account') : submitting ? 'Logging in…' : 'Log in'}
      </button>
    </form>
  )
}

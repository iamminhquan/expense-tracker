import { useState, type ChangeEvent, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Banner } from '../../components/ui/Banner'
import { Field } from '../../components/ui/Field'
import { PasswordInput } from '../../components/ui/PasswordInput'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'

export function RegisterForm() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ name: '', email: '', username: '', password: '', passwordConfirm: '' })
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  function update(field: keyof typeof form) {
    return (e: ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [field]: e.target.value }))
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
    <form onSubmit={onSubmit} className="space-y-4">
      {error && <Banner kind="danger">{error}</Banner>}
      <Field label="Name">
        {(control) => <input {...control} required autoComplete="name" className={inputClass} value={form.name} onChange={update('name')} />}
      </Field>
      <Field label="Email">
        {(control) => <input {...control} type="email" required autoComplete="email" className={inputClass} value={form.email} onChange={update('email')} />}
      </Field>
      <Field label="Username" hint="3–20 characters: lowercase letters, numbers or underscores, starting with a letter.">
        {(control) => <input {...control} required autoCapitalize="none" autoComplete="username" className={inputClass} value={form.username} onChange={update('username')} />}
      </Field>
      <Field label="Password">
        {(control) => <PasswordInput {...control} required autoComplete="new-password" value={form.password} onChange={update('password')} />}
      </Field>
      <Field label="Confirm password">
        {(control) => <PasswordInput {...control} required autoComplete="new-password" value={form.passwordConfirm} onChange={update('passwordConfirm')} />}
      </Field>
      <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary', 'lg')} w-full`}>
        {submitting ? 'Creating account…' : 'Create account'}
      </button>
    </form>
  )
}

import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { AuthLayout } from '../../components/layout/AuthLayout'
import { Field } from '../../components/ui/Field'
import { StatusIcon } from '../../components/ui/StatusIcon'
import { authTitleClass, buttonClass, inputClass } from '../../lib/formStyles'
import { forgotPassword } from '../../lib/api/auth'


export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    try {
      // The response never says whether the account exists, so there's no error to show.
      await forgotPassword(email)
    } finally {
      setSubmitting(false)
      setSent(true)
    }
  }

  return (
    <AuthLayout>
      {sent ? (
        <div className="space-y-3 text-center" role="status">
          <StatusIcon kind="mail" />
          <h1 className={authTitleClass}>Check your inbox</h1>
          <p className="text-[15px] leading-[22px] text-ink-muted">
            If <span className="font-semibold break-all text-ink">{email}</span> has an account, a reset link is on its way. It works for one hour.
          </p>
        </div>
      ) : (
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <h1 className={authTitleClass}>Reset your password</h1>
            <p className="mt-1.5 text-[15px] leading-[22px] text-ink-muted">Enter your email and we'll send you a link to choose a new one.</p>
          </div>
          <Field label="Email">
            {(control) => <input {...control} type="email" required autoComplete="email" className={inputClass} value={email} onChange={(e) => setEmail(e.target.value)} />}
          </Field>
          <button type="submit" disabled={submitting} aria-busy={submitting} className={`${buttonClass('primary')} w-full`}>
            {submitting ? 'Sending…' : 'Send reset link'}
          </button>
        </form>
      )}
      <Link to="/login" className={`${buttonClass('ghost')} w-full`}>
        <ChevronLeft aria-hidden="true" />
        Back to log in
      </Link>
    </AuthLayout>
  )
}

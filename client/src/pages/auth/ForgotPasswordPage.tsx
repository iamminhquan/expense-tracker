import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { AuthLayout, inputClass, primaryButtonClass } from '../../components/layout/AuthLayout'
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
        <p className="text-center text-[14px] text-ink-muted">
          If <span className="font-medium text-ink">{email}</span> has an account, a reset link is on its way.
        </p>
      ) : (
        <form onSubmit={onSubmit} className="space-y-4">
          <p className="text-[14px] text-ink-muted">Enter your email and we'll send you a link to reset your password.</p>
          <div>
            <label className="mb-1 block text-[13px] text-ink-muted" htmlFor="email">
              Email
            </label>
            <input id="email" type="email" required className={inputClass} value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <button type="submit" disabled={submitting} className={primaryButtonClass}>
            {submitting ? 'Sending…' : 'Send reset link'}
          </button>
        </form>
      )}
      <p className="mt-4 text-center text-[13px] text-ink-faint">
        <Link to="/login" className="text-accent hover:underline">
          Back to log in
        </Link>
      </p>
    </AuthLayout>
  )
}

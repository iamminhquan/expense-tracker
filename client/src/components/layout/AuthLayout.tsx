import type { ReactNode } from 'react'

/** The centered card shell every pre-auth page (login/register, forgot/reset-password, verify-email) shares. */
export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-app px-4">
      <div className="w-full max-w-[380px]">
        <div className="mb-8 text-center">
          <span role="img" aria-label="$pend — Expenses" className="wordmark text-[34px]">
            $pend
          </span>
        </div>
        <div className="rounded-[20px] border border-border-card bg-surface p-6 shadow-sm">{children}</div>
      </div>
    </div>
  )
}

export function FieldError({ children }: { children?: string | null }) {
  if (!children) return null
  return <p className="mt-2 text-[13px] text-expense">{children}</p>
}

export const inputClass =
  'w-full rounded-[10px] border border-border-input bg-surface px-3 py-2 text-[14px] text-ink placeholder:text-placeholder focus:outline-2 focus:outline-accent/40'

export const primaryButtonClass =
  'w-full rounded-[10px] bg-accent px-4 py-2.5 text-[14px] font-semibold text-on-solid hover:opacity-90 disabled:opacity-50'

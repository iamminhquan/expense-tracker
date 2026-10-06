import type { ReactNode } from 'react'

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

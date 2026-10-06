import type { ReactNode } from 'react'

export function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center bg-app px-4 pt-8 pb-10 sm:px-6 sm:pt-16 sm:pb-[72px]">
      <p className="mb-6 sm:mb-8">
        <span role="img" aria-label="$pend" className="wordmark text-[30px] leading-9 sm:text-[32px]">
          $pend
        </span>
      </p>
      <main className="w-full max-w-[440px] animate-page-in space-y-5 rounded-[28px] border border-border bg-surface p-6 sm:p-9">{children}</main>
    </div>
  )
}

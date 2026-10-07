import { useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { ChartPie, Plus, ReceiptText, Settings, Tags } from 'lucide-react'
import { useDashboard } from '../../hooks/useDashboard'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { AddTransactionSheet } from '../AddTransactionSheet'
import { ErrorBoundary } from '../ErrorBoundary'
import { Wordmark } from '../ui/Wordmark'
import { BalanceWidget } from './BalanceWidget'
import { UserMenu } from './UserMenu'
import { VerifyEmailBanner } from './VerifyEmailBanner'

const NAV_LINKS = [
  { to: '/dashboard', label: 'Overview', Icon: ChartPie },
  { to: '/transactions', label: 'Transactions', Icon: ReceiptText },
  { to: '/categories', label: 'Categories', Icon: Tags },
  { to: '/settings', label: 'Settings', Icon: Settings },
]

function DockLink({ to, label, Icon }: (typeof NAV_LINKS)[number]) {
  return (
    <NavLink to={to} className="group flex h-14 min-w-0 flex-1 flex-col items-center justify-center gap-1 rounded-[20px] text-[11px] leading-[14px] font-semibold [font-stretch:92%]">
      {({ isActive }) => (
        <>
          <span
            aria-hidden="true"
            className={`flex h-7 w-12 items-center justify-center rounded-full ${
              isActive ? 'bg-accent-tint text-expense' : 'text-ink-muted group-hover:bg-surface-2 group-hover:text-ink'
            }`}
          >
            <Icon className="size-[21px]" strokeWidth={isActive ? 2.3 : 2} />
          </span>
          <span className={`max-w-full truncate ${isActive ? 'text-ink' : 'text-ink-muted'}`}>{label}</span>
        </>
      )}
    </NavLink>
  )
}

export function Layout() {
  // Fetched once here, so every page shares the one headerBalance request.
  const { data: dashboard } = useDashboard()
  const location = useLocation()
  const isDesktop = useIsDesktop()
  const [adding, setAdding] = useState(false)

  return (
    <div className="min-h-screen bg-app text-ink">
      <header className="sticky top-0 z-40 hidden h-[72px] border-b border-border bg-app/85 backdrop-blur-xl md:block">
        <div className="mx-auto flex h-full w-full max-w-[1280px] items-center gap-8 px-8 lg:px-10">
          <Wordmark className="shrink-0 text-[26px]" />
          <nav aria-label="Main" className="flex items-center gap-1">
            {NAV_LINKS.slice(0, 3).map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  `flex h-10 items-center rounded-full px-4 text-[14px] leading-5 font-semibold ${
                    isActive ? 'bg-ink text-app' : 'text-ink-muted hover:bg-surface-2 hover:text-ink'
                  }`
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-5">
            {dashboard && <BalanceWidget balance={dashboard.headerBalance} />}
            <UserMenu />
          </div>
        </div>
      </header>

      <header className="sticky top-0 z-50 flex h-16 items-center justify-between gap-3 bg-app/85 px-4 backdrop-blur-xl md:hidden">
        <Wordmark className="text-[24px]" />
        <div className="flex min-w-0 items-center gap-3">
          {dashboard && <BalanceWidget balance={dashboard.headerBalance} compact />}
          <UserMenu />
        </div>
      </header>

      <main className="mx-auto w-full max-w-[1280px] px-4 pt-3 pb-[var(--dock-clearance)] md:px-8 md:pt-10 md:pb-20 lg:px-10">
        <VerifyEmailBanner />
        <div key={location.pathname} className="animate-page-in">
          <ErrorBoundary resetKey={location.pathname}>
            <Outlet />
          </ErrorBoundary>
        </div>
      </main>

      <nav
        aria-label="Main"
        className="fixed inset-x-3 bottom-[max(12px,env(safe-area-inset-bottom))] z-40 flex items-center rounded-[28px] border border-border bg-surface/95 p-1.5 shadow-float backdrop-blur-xl md:hidden"
      >
        <DockLink {...NAV_LINKS[0]} />
        <DockLink {...NAV_LINKS[1]} />
        <div className="flex w-[72px] shrink-0 justify-center">
          <button
            type="button"
            onClick={() => setAdding(true)}
            aria-label="Add a transaction"
            aria-haspopup="dialog"
            className="press -mt-7 flex size-[60px] items-center justify-center rounded-full bg-accent text-on-accent shadow-accent ring-[5px] ring-app hover:bg-accent-hover"
          >
            <Plus aria-hidden="true" strokeWidth={2.6} className="size-7" />
          </button>
        </div>
        <DockLink {...NAV_LINKS[2]} />
        <DockLink {...NAV_LINKS[3]} />
      </nav>

      {!isDesktop && <AddTransactionSheet open={adding} onClose={() => setAdding(false)} />}
    </div>
  )
}

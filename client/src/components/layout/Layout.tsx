import { useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { ArrowLeftRight, ChartPie, Plus, Tags } from 'lucide-react'
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
  { to: '/transactions', label: 'Transactions', Icon: ArrowLeftRight },
  { to: '/categories', label: 'Categories', Icon: Tags },
]

export function Layout() {
  // Fetched once here, so every page shares the one headerBalance request.
  const { data: dashboard } = useDashboard()
  const location = useLocation()
  const isDesktop = useIsDesktop()
  const [addOpen, setAddOpen] = useState(false)
  // Mounted on the first tap, so the categories request isn't made on every mobile page load.
  const [addMounted, setAddMounted] = useState(false)

  return (
    <div className="min-h-screen bg-app text-ink">
      <header className="sticky top-0 z-40 hidden h-[72px] border-b border-border bg-surface/85 backdrop-blur-md md:block">
        <div className="mx-auto flex h-full w-full max-w-[1280px] items-center gap-8 px-10">
          <Wordmark className="shrink-0 text-[30px] leading-8" />
          <nav aria-label="Main" className="flex items-center gap-1">
            {NAV_LINKS.map(({ to, label, Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  `flex h-10 items-center gap-2 rounded-full px-4 text-[14px] leading-5 font-semibold ${
                    isActive ? 'bg-accent-tint text-accent-text' : 'text-ink-muted hover:bg-surface-2 hover:text-ink'
                  }`
                }
              >
                <Icon aria-hidden="true" className="size-[18px]" />
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-4">
            {dashboard && (
              <div className="rounded-full border border-border bg-app py-1.5 pr-6 pl-2">
                <BalanceWidget balance={dashboard.headerBalance} />
              </div>
            )}
            <UserMenu />
          </div>
        </div>
      </header>

      <header className="sticky top-0 z-50 flex h-16 items-center justify-between bg-app/90 px-4 backdrop-blur-md md:hidden">
        <Wordmark className="text-[30px] leading-8" />
        <UserMenu balance={dashboard?.headerBalance} />
      </header>

      <main className="mx-auto w-full max-w-[1280px] px-4 pt-3 pb-[136px] md:px-10 md:pt-10 md:pb-20">
        <VerifyEmailBanner />
        <div key={location.pathname} className="animate-page-in">
          <ErrorBoundary resetKey={location.pathname}>
            <Outlet />
          </ErrorBoundary>
        </div>
      </main>

      <div className="pointer-events-none fixed inset-x-0 bottom-0 z-40 flex items-center gap-3 px-4 pb-[max(1rem,env(safe-area-inset-bottom))] md:hidden">
        <nav
          aria-label="Main"
          className="pointer-events-auto flex h-16 flex-1 gap-1 rounded-full border border-border bg-surface/95 p-1.5 shadow-float backdrop-blur-md"
        >
          {NAV_LINKS.map(({ to, label, Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex flex-1 flex-col items-center justify-center gap-0.5 rounded-full text-[12px] leading-4 font-semibold active:scale-95 motion-reduce:active:scale-100 ${
                  isActive ? 'bg-accent-tint text-accent-text' : 'text-ink-muted hover:text-ink'
                }`
              }
            >
              <Icon aria-hidden="true" className="size-[22px]" />
              {label}
            </NavLink>
          ))}
        </nav>
        <button
          type="button"
          onClick={() => {
            setAddMounted(true)
            setAddOpen(true)
          }}
          aria-label="Add transaction"
          className="pointer-events-auto flex size-16 shrink-0 items-center justify-center rounded-full bg-accent text-on-accent shadow-float transition-transform duration-150 active:scale-90 motion-reduce:active:scale-100"
        >
          <Plus aria-hidden="true" strokeWidth={2.5} className="size-8" />
        </button>
      </div>

      {!isDesktop && addMounted && <AddTransactionSheet open={addOpen} onClose={() => setAddOpen(false)} />}
    </div>
  )
}

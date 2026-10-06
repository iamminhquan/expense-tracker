import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { ArrowLeftRight, ChartPie, Tags } from 'lucide-react'
import { useDashboard } from '../../hooks/useDashboard'
import { ErrorBoundary } from '../ErrorBoundary'
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

  return (
    <div className="min-h-screen bg-app text-ink">
      <header className="sticky top-0 z-40 hidden h-16 border-b border-border bg-surface md:block">
        <div className="mx-auto flex h-full w-full max-w-[1280px] items-center gap-10 px-10">
          <span className="wordmark shrink-0 text-[28px] leading-8">$pend</span>
          <nav aria-label="Main" className="flex h-full items-center gap-7">
            {NAV_LINKS.map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  `relative flex h-full items-center text-[14px] leading-5 font-semibold after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 ${
                    isActive ? 'text-ink after:bg-accent' : 'text-ink-muted hover:text-ink'
                  }`
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-6">
            {dashboard && (
              <div className="border-l border-border pl-6">
                <BalanceWidget balance={dashboard.headerBalance} />
              </div>
            )}
            <UserMenu />
          </div>
        </div>
      </header>

      <header className="sticky top-0 z-50 flex h-16 items-center justify-between bg-app px-4 md:hidden">
        <span className="wordmark text-[28px] leading-8">$pend</span>
        <UserMenu balance={dashboard?.headerBalance} />
      </header>

      <main className="mx-auto w-full max-w-[1280px] px-4 pt-2 pb-[120px] md:px-10 md:pt-9 md:pb-16">
        <VerifyEmailBanner />
        <div key={location.pathname} className="animate-page-in">
          <ErrorBoundary resetKey={location.pathname}>
            <Outlet />
          </ErrorBoundary>
        </div>
      </main>

      <nav
        aria-label="Main"
        className="fixed inset-x-4 bottom-4 z-40 flex gap-1 rounded-[34px] border border-border bg-surface p-2 shadow-float md:hidden"
      >
        {NAV_LINKS.map(({ to, label, Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex h-[52px] flex-1 flex-col items-center justify-center gap-0.5 rounded-[26px] text-[12px] leading-4 font-semibold ${
                isActive ? 'bg-accent text-on-accent' : 'text-ink-muted hover:bg-surface-2 hover:text-ink'
              }`
            }
          >
            <Icon aria-hidden="true" className="size-[22px]" />
            {label}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}

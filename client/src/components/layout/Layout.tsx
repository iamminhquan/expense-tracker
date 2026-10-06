import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { ErrorBoundary } from '../ErrorBoundary'
import { useDashboard } from '../../hooks/useDashboard'
import { BalanceWidget } from './BalanceWidget'
import { UserMenu } from './UserMenu'

const NAV_LINKS = [
  { to: '/dashboard', label: 'Overview' },
  { to: '/transactions', label: 'Transactions' },
  { to: '/categories', label: 'Categories' },
]

export function Layout() {
  const { pathname } = useLocation()
  // Fetched once here, so every page shares the one headerBalance request.
  const { data: dashboard } = useDashboard()

  return (
    <div className="min-h-screen bg-app text-ink">
      <nav className="sticky top-0 z-40 hidden h-[60px] items-center border-b border-border-nav bg-surface md:flex">
        <div className="mx-auto flex h-full w-full max-w-[1280px] items-center gap-10 px-9">
          <span className="wordmark shrink-0 text-[19px]">$pend</span>
          <div className="flex items-center gap-2">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                className={({ isActive }) =>
                  `rounded-[7px] px-[16px] py-[7px] text-[13px] ${
                    isActive ? 'bg-accent/10 font-semibold text-accent' : 'text-nav-idle hover:bg-track hover:text-ink'
                  }`
                }
              >
                {link.label}
              </NavLink>
            ))}
          </div>
          <div className="ml-auto flex items-center gap-3">
            {dashboard && <BalanceWidget balance={dashboard.headerBalance} />}
            <UserMenu />
          </div>
        </div>
      </nav>

      <header className="sticky top-0 z-40 flex h-14 items-center justify-between border-b border-border-nav bg-surface px-4 md:hidden">
        <span className="wordmark text-[18px]">$pend</span>
        <div className="flex items-center gap-2">
          {dashboard && <BalanceWidget balance={dashboard.headerBalance} />}
          <UserMenu />
        </div>
      </header>

      <main className="mx-auto w-full max-w-[1280px] px-4 py-6 md:px-9">
        <ErrorBoundary resetKey={pathname}>
          <Outlet />
        </ErrorBoundary>
      </main>

      <nav className="sticky bottom-4 z-40 mx-4 mb-4 mt-[14px] flex rounded-[20px] border border-border-card bg-surface p-[6px] shadow-lg md:hidden">
        {NAV_LINKS.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center justify-center gap-[3px] rounded-[14px] py-2 ${isActive ? 'bg-accent/10' : ''}`
            }
          >
            {({ isActive }) => (
              <span className={`text-[11px] ${isActive ? 'font-semibold text-accent' : 'text-ink-faint'}`}>{link.label}</span>
            )}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}

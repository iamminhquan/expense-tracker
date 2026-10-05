import { NavLink, Outlet } from 'react-router-dom'
import { useDashboard } from '../../hooks/useDashboard'
import { BalanceWidget } from './BalanceWidget'
import { UserMenu } from './UserMenu'

const NAV_LINKS = [
  { to: '/dashboard', label: 'Overview' },
  { to: '/transactions', label: 'Transactions' },
  { to: '/categories', label: 'Categories' },
]

// Mirrors nav.html: both bars exist at once, each hidden at the breakpoint
// the other owns via Tailwind's md: prefix, rather than mounting/unmounting
// one on resize.
export function Layout() {
  // headerBalance is always the real current month regardless of which
  // page is showing (see dashboard_handlers.go's comment) -- fetching it
  // through useDashboard() here means every protected page shares the one
  // cached request instead of each rolling its own.
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
        <Outlet />
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

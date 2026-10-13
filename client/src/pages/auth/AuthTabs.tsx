import { Link } from 'react-router-dom'

const TABS = [
  { to: '/login', key: 'login', label: 'Log in' },
  { to: '/register', key: 'register', label: 'Sign up' },
] as const

export function AuthTabs({ active }: { active: 'login' | 'register' }) {
  return (
    <nav aria-label="Log in or sign up" className="relative grid grid-cols-2 rounded-control bg-surface-2 p-1">
      <span
        aria-hidden="true"
        className={`tab-thumb absolute inset-y-1 left-1 w-[calc(50%-4px)] rounded-[10px] bg-surface shadow-card ${active === 'register' ? 'translate-x-full' : ''}`}
      />
      {TABS.map((tab) => {
        const selected = tab.key === active
        return (
          <Link
            key={tab.key}
            to={tab.to}
            aria-current={selected ? 'page' : undefined}
            className={`relative flex h-11 items-center justify-center rounded-[10px] text-[15px] leading-5 font-semibold ${
              selected ? 'text-ink' : 'text-ink-muted hover:text-ink'
            }`}
          >
            {tab.label}
          </Link>
        )
      })}
    </nav>
  )
}

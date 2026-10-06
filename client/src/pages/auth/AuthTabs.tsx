import { Link } from 'react-router-dom'

const TABS = [
  { to: '/login', key: 'login', label: 'Log in' },
  { to: '/register', key: 'register', label: 'Sign up' },
] as const

export function AuthTabs({ active }: { active: 'login' | 'register' }) {
  return (
    <nav aria-label="Log in or sign up" className="flex gap-1 rounded-[16px] bg-surface-2 p-1">
      {TABS.map((tab) => {
        const selected = tab.key === active
        return (
          <Link
            key={tab.key}
            to={tab.to}
            aria-current={selected ? 'page' : undefined}
            className={`flex h-11 flex-1 items-center justify-center rounded-[12px] text-[15px] leading-5 font-semibold ${
              selected ? 'bg-accent text-on-accent' : 'text-ink-muted hover:text-ink'
            }`}
          >
            {tab.label}
          </Link>
        )
      })}
    </nav>
  )
}

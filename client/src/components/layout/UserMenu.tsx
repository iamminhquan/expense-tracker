import { Link } from 'react-router-dom'
import { useAuth } from '../../lib/auth/AuthContext'
import { useTheme } from '../../lib/theme/ThemeContext'
import type { Theme } from '../../lib/api/types'

const THEMES: { value: Theme; label: string }[] = [
  { value: 'light', label: 'Light' },
  { value: 'auto', label: 'Auto' },
  { value: 'dark', label: 'Dark' },
]

/** Mirrors user_menu.html: theme switch, a link to Settings, and logout. */
export function UserMenu() {
  const { user, logout } = useAuth()
  const { theme, setTheme } = useTheme()
  const initial = user?.username ? user.username[0].toUpperCase() : '?'

  return (
    <details className="relative shrink-0">
      <summary className="flex cursor-pointer list-none items-center gap-2 rounded-[7px] px-2 py-1 hover:bg-track [&::-webkit-details-marker]:hidden">
        <span className="flex h-[26px] w-[26px] items-center justify-center rounded-full bg-border-nav text-[11px] font-semibold text-ink-muted md:h-6 md:w-6">
          {initial}
        </span>
        <span className="text-[13px] text-ink-muted">{user?.name}</span>
      </summary>
      <div className="absolute right-0 top-full z-50 mt-1 w-[220px] rounded-[14px] border border-border-card bg-surface p-3 shadow-lg">
        <p className="mb-2 text-[10px] font-semibold tracking-[0.08em] text-ink-faintest">APPEARANCE</p>
        <div className="mb-3 flex gap-1 rounded-[9px] bg-track p-[3px]">
          {THEMES.map((t) => (
            <button
              key={t.value}
              type="button"
              aria-pressed={theme === t.value}
              onClick={() => setTheme(t.value)}
              className={`h-7 flex-1 rounded-[6px] text-[12px] ${
                theme === t.value ? 'bg-accent/10 font-semibold text-accent' : 'text-ink-faint'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
        <Link to="/settings" className="flex w-full items-center gap-2 rounded-[8px] px-2 py-2 text-[13px] text-ink-muted hover:bg-track">
          Settings
        </Link>
        <div className="my-2 h-px bg-border-card" />
        <button
          type="button"
          onClick={() => void logout()}
          className="flex w-full items-center gap-2 whitespace-nowrap rounded-[8px] px-2 py-2 text-[13px] text-expense hover:bg-expense/10"
        >
          Log out
        </button>
      </div>
    </details>
  )
}

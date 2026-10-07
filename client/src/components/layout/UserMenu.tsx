import { useCallback, useId, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { LogOut, Monitor, Moon, Settings, Sun } from 'lucide-react'
import { useAuth } from '../../lib/auth/AuthContext'
import { useTheme } from '../../lib/theme/ThemeContext'
import { useDismiss } from '../../hooks/useDismiss'
import { SegmentedControl } from '../ui/SegmentedControl'
import type { Balance, Theme } from '../../lib/api/types'
import { BalanceWidget } from './BalanceWidget'

const rowClass = 'flex min-h-12 w-full items-center gap-3 rounded-control px-3 text-[15px] leading-5 font-semibold [&_svg]:size-5'

interface UserMenuProps {
  /** Mobile only: the header has no room for the balance, so the menu carries it. */
  balance?: Balance
}

export function UserMenu({ balance }: UserMenuProps) {
  const { user, logout } = useAuth()
  const { theme, setTheme } = useTheme()
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const panelId = useId()
  const close = useCallback(() => setOpen(false), [])
  useDismiss(open, panelRef, close, buttonRef)

  const initial = (user?.name || user?.username || '?')[0].toUpperCase()

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        aria-label={`Account menu for ${user?.name ?? 'you'}`}
        onClick={() => setOpen((o) => !o)}
        className={`flex size-11 items-center justify-center rounded-full border font-display text-[18px] font-bold active:scale-95 motion-reduce:active:scale-100 ${
          open ? 'border-accent-text bg-accent text-on-accent' : 'border-transparent bg-accent-tint text-accent-text hover:border-accent-text/40'
        }`}
      >
        {initial}
      </button>

      {open && (
        <>
          <div aria-hidden="true" className="fixed inset-0 top-16 z-40 animate-fade-in bg-scrim md:hidden" />
          <div
            ref={panelRef}
            id={panelId}
            className="fixed inset-x-4 top-[68px] z-50 animate-pop-in rounded-card border border-border bg-surface p-2 shadow-popover md:absolute md:inset-x-auto md:top-[calc(100%+12px)] md:right-0 md:w-80"
          >
            <div className="flex items-center gap-3 px-3 pt-2.5 pb-3.5">
              <span aria-hidden="true" className="flex size-11 shrink-0 items-center justify-center rounded-full bg-accent-tint font-display text-[18px] font-bold text-accent-text">
                {initial}
              </span>
              <div className="min-w-0">
                <p className="truncate text-[15px] leading-5 font-semibold text-ink">{user?.name}</p>
                <p className="truncate text-[13px] leading-[18px] text-ink-muted">{user?.email}</p>
              </div>
            </div>
            {balance && (
              <div className="mx-1 mb-2 rounded-[18px] bg-surface-2 p-3.5 md:hidden">
                <BalanceWidget balance={balance} />
              </div>
            )}
            <Link to="/settings" onClick={close} className={`${rowClass} text-ink hover:bg-surface-2`}>
              <Settings aria-hidden="true" />
              Settings
            </Link>
            <div className="px-1 py-2">
              <p className="mb-2 px-2 text-[13px] leading-[18px] font-semibold text-ink-muted">Appearance</p>
              <SegmentedControl<Theme>
                label="Theme"
                value={theme}
                onChange={setTheme}
                size="tall"
                fullWidth
                options={[
                  { value: 'light', label: 'Light', icon: <Sun aria-hidden="true" /> },
                  { value: 'auto', label: 'Auto', icon: <Monitor aria-hidden="true" /> },
                  { value: 'dark', label: 'Dark', icon: <Moon aria-hidden="true" /> },
                ]}
              />
            </div>
            <div className="mx-2 my-1 h-px bg-border" />
            <button type="button" onClick={() => void logout()} className={`${rowClass} text-danger hover:bg-danger-tint`}>
              <LogOut aria-hidden="true" />
              Log out
            </button>
          </div>
        </>
      )}
    </div>
  )
}

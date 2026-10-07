import { useCallback, useId, useRef, useState, type KeyboardEvent } from 'react'
import { Calendar, Check, ChevronDown } from 'lucide-react'
import { useDismiss } from '../hooks/useDismiss'
import type { MonthOption } from '../lib/api/types'

interface MonthPickerProps {
  value: string
  label: string
  currentMonthValue: string
  availableMonths: MonthOption[]
  onChange: (value: string) => void
  /** Only the transactions page offers "All months"; the dashboard never does. */
  allowAllMonths?: boolean
  size?: 'md' | 'lg'
}

export function MonthPicker({ value, label, currentMonthValue, availableMonths, onChange, allowAllMonths, size = 'md' }: MonthPickerProps) {
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const listRef = useRef<HTMLUListElement>(null)
  const listId = useId()
  const close = useCallback(() => setOpen(false), [])
  useDismiss(open, listRef, close, buttonRef)

  const options: MonthOption[] = [
    ...(allowAllMonths ? [{ value: 'all', label: 'All months' }] : []),
    ...(availableMonths.some((m) => m.value === currentMonthValue) ? [] : [{ value: currentMonthValue, label: value === currentMonthValue ? label : 'This month' }]),
    ...availableMonths,
  ]

  function openList() {
    setActive(Math.max(0, options.findIndex((o) => o.value === value)))
    setOpen(true)
    requestAnimationFrame(() => listRef.current?.focus())
  }

  function choose(option: MonthOption) {
    setOpen(false)
    buttonRef.current?.focus()
    if (option.value !== value) onChange(option.value)
  }

  function onListKeyDown(e: KeyboardEvent<HTMLUListElement>) {
    const last = options.length - 1
    const moves: Record<string, number> = { ArrowDown: Math.min(last, active + 1), ArrowUp: Math.max(0, active - 1), Home: 0, End: last }
    if (e.key in moves) {
      e.preventDefault()
      setActive(moves[e.key])
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      choose(options[active])
    } else if (e.key === 'Tab') {
      setOpen(false)
    }
  }

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        aria-label={`Month: ${label}`}
        onClick={() => (open ? setOpen(false) : openList())}
        onKeyDown={(e) => {
          if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            e.preventDefault()
            openList()
          }
        }}
        className={`press inline-flex items-center gap-2 rounded-full border pr-3.5 pl-4 text-[14px] leading-5 font-semibold text-ink ${
          size === 'lg' ? 'h-12' : 'h-11'
        } ${open ? 'border-ink bg-surface-2' : 'border-border-strong/45 bg-surface hover:border-border-strong hover:bg-surface-2'}`}
      >
        <Calendar aria-hidden="true" className="size-[18px] text-expense" />
        <span className="tabular whitespace-nowrap">{label}</span>
        <ChevronDown aria-hidden="true" className={`size-4 text-ink-muted transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <ul
          ref={listRef}
          id={listId}
          role="listbox"
          tabIndex={-1}
          aria-label="Month"
          aria-activedescendant={`${listId}-${active}`}
          onKeyDown={onListKeyDown}
          className="fixed inset-x-3 bottom-[var(--dock-clearance)] z-50 max-h-[50dvh] animate-pop-in overflow-y-auto rounded-[22px] border border-border bg-surface p-1.5 shadow-popover focus-visible:outline-none sm:absolute sm:inset-x-auto sm:top-[calc(100%+8px)] sm:right-0 sm:bottom-auto sm:max-h-[340px] sm:w-60"
        >
          {options.map((option, i) => {
            const selected = option.value === value
            return (
              <li
                key={option.value}
                id={`${listId}-${i}`}
                role="option"
                aria-selected={selected}
                onClick={() => choose(option)}
                onPointerMove={() => setActive(i)}
                className={`flex min-h-11 cursor-pointer items-center justify-between rounded-[14px] px-3 text-[15px] leading-5 ${
                  selected ? 'font-semibold text-expense' : 'text-ink'
                } ${i === active ? 'bg-surface-2 outline-2 outline-accent -outline-offset-2' : ''}`}
              >
                {option.label}
                {selected && <Check aria-hidden="true" className="size-[18px]" />}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

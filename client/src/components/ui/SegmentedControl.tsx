import type { ReactNode } from 'react'

interface SegmentedOption<T extends string> {
  value: T
  label: ReactNode
  icon?: ReactNode
}

interface SegmentedControlProps<T extends string> {
  label: string
  value: T
  options: SegmentedOption<T>[]
  onChange: (value: T) => void
  /** Tall is the 52px variant the mobile type toggle and the theme switch use. */
  size?: 'md' | 'tall'
  fullWidth?: boolean
}

export function SegmentedControl<T extends string>({ label, value, options, onChange, size = 'md', fullWidth }: SegmentedControlProps<T>) {
  return (
    <div
      role="group"
      aria-label={label}
      className={`${fullWidth ? 'flex w-full' : 'inline-flex'} gap-1 rounded-[14px] bg-surface-2 p-1`}
    >
      {options.map((option) => {
        const selected = option.value === value
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={selected}
            onClick={() => onChange(option.value)}
            className={`inline-flex items-center justify-center gap-1.5 rounded-[10px] text-[14px] leading-5 font-semibold [&_svg]:size-4 ${
              size === 'tall' ? 'h-11' : 'h-9'
            } ${fullWidth ? 'flex-1 px-2' : 'px-[14px]'} ${
              selected ? 'bg-accent text-on-accent' : 'text-ink-muted hover:bg-[color-mix(in_srgb,var(--color-surface-2),var(--color-ink)_8%)] hover:text-ink'
            }`}
          >
            {option.icon}
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

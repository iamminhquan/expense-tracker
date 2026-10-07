export type ButtonVariant = 'primary' | 'income' | 'secondary' | 'tonal' | 'ghost' | 'danger' | 'danger-tint' | 'danger-ghost'
export type ButtonSize = 'lg' | 'md' | 'sm'

const buttonBase =
  'press inline-flex shrink-0 items-center justify-center gap-2 rounded-full font-semibold whitespace-nowrap select-none ' +
  'disabled:cursor-not-allowed disabled:border-transparent disabled:bg-surface-2 disabled:text-ink-muted disabled:shadow-none'

const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-on-accent hover:bg-accent-hover',
  income: 'bg-income text-on-income hover:bg-[color-mix(in_srgb,var(--color-income)_86%,var(--color-ink))]',
  secondary: 'border border-border-strong/45 bg-surface text-ink hover:border-border-strong hover:bg-surface-2',
  tonal: 'bg-accent-tint text-expense hover:bg-[color-mix(in_srgb,var(--color-accent-tint),var(--color-accent)_12%)]',
  ghost: 'text-ink hover:bg-surface-2',
  danger: 'bg-danger text-on-danger hover:bg-[color-mix(in_srgb,var(--color-danger)_86%,var(--color-ink))]',
  'danger-tint': 'bg-danger-tint text-danger hover:bg-[color-mix(in_srgb,var(--color-danger-tint),var(--color-danger)_12%)]',
  'danger-ghost': 'text-danger hover:bg-danger-tint',
}

// Small buttons stay 44px tall on touch screens and only shrink where a mouse is likely.
const buttonSizes: Record<ButtonSize, string> = {
  lg: 'h-[52px] px-6 text-[16px] leading-5 [&_svg]:size-5',
  md: 'h-11 px-5 text-[15px] leading-5 [&_svg]:size-[18px]',
  sm: 'h-11 px-4 text-[14px] leading-5 md:h-9 md:px-3.5 [&_svg]:size-4',
}

export function buttonClass(variant: ButtonVariant = 'primary', size: ButtonSize = 'md'): string {
  return `${buttonBase} ${buttonVariants[variant]} ${buttonSizes[size]}`
}

export const iconButtonClass =
  'press inline-flex size-11 shrink-0 items-center justify-center rounded-full text-ink-muted hover:bg-surface-2 hover:text-ink ' +
  'disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-5'

const controlBase =
  'h-12 w-full border border-border-strong/70 bg-surface text-[16px] leading-6 font-medium text-ink ' +
  'hover:border-border-strong focus-visible:border-accent focus-visible:outline-offset-0 disabled:cursor-not-allowed disabled:border-border disabled:bg-surface-2 ' +
  'disabled:text-ink-muted aria-invalid:border-danger aria-invalid:shadow-[0_0_0_1px_var(--color-danger)]'

export const inputClass = `${controlBase} rounded-control px-4`

// The transactions search is a pill like the buttons beside it, with room for its icon.
export const searchInputClass = `${controlBase} rounded-full pr-4 pl-11`

export const selectClass = `${controlBase} rounded-control cursor-pointer appearance-none pr-10 pl-4`

export const labelClass = 'mb-2 block text-[13px] leading-[18px] font-semibold text-ink'

/* A selectable pill for category and date choices. The tone follows the transaction type being entered. */
export function chipClass(selected: boolean, tone: 'expense' | 'income' = 'expense', padding = 'px-4'): string {
  const state = !selected
    ? 'border-border bg-surface text-ink hover:border-border-strong'
    : tone === 'expense'
      ? 'border-accent bg-accent-tint text-expense'
      : 'border-income bg-income-tint text-income'
  return `press inline-flex min-h-11 items-center gap-2 rounded-full border ${padding} text-[14px] leading-5 font-semibold ${state}`
}

export const cardClass = 'rounded-tile border border-border bg-surface p-5 md:p-7'

export const cardTitleClass = 'heading text-[18px] leading-6 text-ink'

export const pageTitleClass = 'heading text-[30px] leading-[36px] text-ink md:text-[40px] md:leading-[46px]'

export const mutedLabelClass = 'text-[13px] leading-[18px] font-medium text-ink-muted'

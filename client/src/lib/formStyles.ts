export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'danger-tint' | 'danger-ghost'
export type ButtonSize = 'md' | 'sm'

const buttonBase =
  'inline-flex shrink-0 items-center justify-center gap-2 rounded-[12px] font-semibold whitespace-nowrap select-none ' +
  'active:scale-[.98] motion-reduce:active:scale-100 disabled:cursor-not-allowed disabled:border-transparent ' +
  'disabled:bg-surface-2 disabled:text-ink-muted disabled:active:scale-100'

const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-on-accent hover:bg-[color-mix(in_srgb,var(--color-accent)_82%,var(--color-app))]',
  secondary: 'border border-border-strong bg-surface text-ink hover:bg-surface-2',
  ghost: 'text-ink hover:bg-surface-2',
  danger: 'bg-danger text-on-danger hover:bg-[color-mix(in_srgb,var(--color-danger)_88%,var(--color-ink))]',
  'danger-tint': 'bg-danger-tint text-danger hover:bg-[color-mix(in_srgb,var(--color-danger-tint),var(--color-danger)_10%)]',
  'danger-ghost': 'text-danger hover:bg-danger-tint',
}

const buttonSizes: Record<ButtonSize, string> = {
  md: 'h-11 px-[18px] text-[15px] leading-5 [&_svg]:size-[18px]',
  sm: 'h-9 px-[14px] text-[14px] leading-5 [&_svg]:size-4',
}

export function buttonClass(variant: ButtonVariant = 'primary', size: ButtonSize = 'md'): string {
  return `${buttonBase} ${buttonVariants[variant]} ${buttonSizes[size]}`
}

export const iconButtonClass =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-[12px] text-ink-muted hover:bg-surface-2 hover:text-ink ' +
  'disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-5'

const controlBase =
  'h-11 w-full rounded-[12px] border border-border-strong bg-surface px-[14px] text-[15px] leading-6 font-medium text-ink ' +
  'hover:border-ink-muted focus-visible:border-accent disabled:cursor-not-allowed disabled:border-border disabled:bg-surface-2 ' +
  'disabled:text-ink-muted aria-invalid:border-danger aria-invalid:shadow-[0_0_0_1px_var(--color-danger)]'

export const inputClass = controlBase

export const selectClass = `${controlBase} cursor-pointer appearance-none pr-10`

export const labelClass = 'mb-1.5 block text-[13px] leading-[18px] font-semibold text-ink'

export const cardClass = 'rounded-[24px] border border-border bg-surface p-5 md:rounded-[28px] md:p-7'

export const cardTitleClass = 'font-display text-[18px] leading-6 font-bold text-ink'

export const pageTitleClass = 'font-display text-[28px] leading-[34px] font-bold tracking-[-0.025em] text-ink md:text-[34px] md:leading-10'

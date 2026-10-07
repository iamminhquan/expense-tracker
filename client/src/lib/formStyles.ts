export type ButtonVariant = 'primary' | 'secondary' | 'tonal' | 'ghost' | 'danger' | 'danger-tint' | 'danger-ghost'
export type ButtonSize = 'md' | 'sm'

const buttonBase =
  'inline-flex shrink-0 items-center justify-center gap-2 rounded-control font-semibold whitespace-nowrap select-none ' +
  'transition-[background-color,color,border-color,transform] duration-150 active:scale-[.97] motion-reduce:active:scale-100 ' +
  'disabled:cursor-not-allowed disabled:border-transparent disabled:bg-surface-2 disabled:text-ink-muted disabled:active:scale-100'

const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-on-accent hover:bg-[color-mix(in_srgb,var(--color-accent),var(--color-ink)_12%)]',
  secondary: 'border border-border-strong/60 bg-surface text-ink hover:border-border-strong hover:bg-surface-2',
  tonal: 'bg-accent-tint text-accent-text hover:bg-[color-mix(in_srgb,var(--color-accent-tint),var(--color-accent-text)_10%)]',
  ghost: 'text-ink hover:bg-surface-2',
  danger: 'bg-danger text-on-danger hover:bg-[color-mix(in_srgb,var(--color-danger)_88%,var(--color-ink))]',
  'danger-tint': 'bg-danger-tint text-danger hover:bg-[color-mix(in_srgb,var(--color-danger-tint),var(--color-danger)_12%)]',
  'danger-ghost': 'text-danger hover:bg-danger-tint',
}

const buttonSizes: Record<ButtonSize, string> = {
  md: 'h-11 px-[18px] text-[15px] leading-5 [&_svg]:size-[18px]',
  sm: 'h-11 px-[14px] text-[14px] leading-5 md:h-9 [&_svg]:size-4',
}

export function buttonClass(variant: ButtonVariant = 'primary', size: ButtonSize = 'md'): string {
  return `${buttonBase} ${buttonVariants[variant]} ${buttonSizes[size]}`
}

export const iconButtonClass =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-surface-2 hover:text-ink ' +
  'active:scale-95 motion-reduce:active:scale-100 disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-5'

const controlBase =
  'h-11 w-full rounded-control border border-border-strong bg-surface px-3.5 text-[15px] leading-6 font-medium text-ink ' +
  'hover:border-ink-muted focus-visible:border-accent-text focus-visible:outline-[3px] focus-visible:outline-offset-0 ' +
  'focus-visible:outline-accent-text/30 disabled:cursor-not-allowed disabled:border-border disabled:bg-surface-2 ' +
  'disabled:text-ink-muted aria-invalid:border-danger aria-invalid:shadow-[0_0_0_1px_var(--color-danger)]'

export const inputClass = controlBase

export const selectClass = `${controlBase} cursor-pointer appearance-none pr-10`

export const labelClass = 'mb-1.5 block text-[13px] leading-[18px] font-semibold text-ink'

export const cardClass = 'rounded-card border border-border bg-surface p-5 shadow-card md:p-6'

/* A card whose rows run edge to edge; the padding lives on the rows. */
export const listCardClass = 'overflow-hidden rounded-card border border-border bg-surface shadow-card'

export const cardTitleClass = 'font-display text-[19px] leading-6 font-semibold tracking-[-0.015em] text-ink'

export const pageTitleClass = 'font-display text-[30px] leading-9 font-bold tracking-[-0.035em] text-ink md:text-[38px] md:leading-[44px]'

export const mutedTextClass = 'text-[13px] leading-[18px] text-ink-muted'

export const authTitleClass = 'font-display text-[28px] leading-8 font-bold tracking-[-0.03em] text-ink'

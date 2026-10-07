interface CategoryAvatarProps {
  name: string
  /** The category's own colour: user data, so the one place a non-token colour is allowed. */
  color: string
  size?: 'md' | 'sm'
}

export function CategoryAvatar({ name, color, size = 'md' }: CategoryAvatarProps) {
  return (
    <span
      aria-hidden="true"
      className={`flex shrink-0 items-center justify-center font-display font-bold ${size === 'md' ? 'size-11 rounded-control text-[18px]' : 'size-9 rounded-tile text-[15px]'}`}
      style={{
        backgroundColor: `color-mix(in srgb, ${color} 24%, var(--color-surface))`,
        color: `color-mix(in srgb, ${color}, var(--color-ink) 48%)`,
      }}
    >
      {(name.trim()[0] ?? '?').toUpperCase()}
    </span>
  )
}

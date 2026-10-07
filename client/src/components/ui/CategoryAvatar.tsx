interface CategoryAvatarProps {
  name: string
  /** The category's stored swatch, the same hex in both themes, so its letter uses on-swatch. */
  color: string
  size?: 'md' | 'lg'
}

export function CategoryAvatar({ name, color, size = 'md' }: CategoryAvatarProps) {
  return (
    <span
      aria-hidden="true"
      className={`heading flex shrink-0 items-center justify-center text-on-swatch ${size === 'lg' ? 'size-12 rounded-[16px] text-[19px]' : 'size-10 rounded-[14px] text-[16px]'}`}
      style={{ backgroundColor: color }}
    >
      {name.trim().charAt(0).toUpperCase() || '·'}
    </span>
  )
}

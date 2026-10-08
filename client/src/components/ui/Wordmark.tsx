interface WordmarkProps {
  className?: string
  /** On the accent panel the whole mark takes the on-accent colour. */
  onAccent?: boolean
}

export function Wordmark({ className = '', onAccent }: WordmarkProps) {
  return (
    <span role="img" aria-label="$pend" className={`wordmark ${onAccent ? '!text-on-accent' : ''} ${className}`}>
      <span aria-hidden="true" className={onAccent ? '' : 'text-accent-text'}>
        $
      </span>
      <span aria-hidden="true">pend</span>
    </span>
  )
}

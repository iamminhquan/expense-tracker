export function Wordmark({ className = '' }: { className?: string }) {
  return (
    <span role="img" aria-label="$pend" className={`wordmark inline-flex items-baseline ${className}`}>
      <span aria-hidden="true" className="text-accent">
        $
      </span>
      <span aria-hidden="true">pend</span>
    </span>
  )
}

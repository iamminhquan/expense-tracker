export function SkeletonBar({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-skeleton rounded-tile bg-border/70 ${className}`} />
}

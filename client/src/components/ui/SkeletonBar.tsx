export function SkeletonBar({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-skeleton rounded-full bg-surface-2 ${className}`} />
}

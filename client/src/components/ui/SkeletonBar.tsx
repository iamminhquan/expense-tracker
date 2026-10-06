export function SkeletonBar({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-skeleton rounded-[8px] bg-surface-2 ${className}`} />
}


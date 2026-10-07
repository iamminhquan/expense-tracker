import { useEffect, useState } from 'react'
import { SkeletonBar } from './SkeletonBar'

// Waits 150ms before showing anything, so a fast response never flashes a skeleton.
export function PageSkeleton({ label = 'Loading…' }: { label?: string }) {
  const [visible, setVisible] = useState(false)
  useEffect(() => {
    const timer = window.setTimeout(() => setVisible(true), 150)
    return () => window.clearTimeout(timer)
  }, [])

  return (
    <div role="status" aria-busy="true">
      <span className="sr-only">{label}</span>
      {visible && (
        <div className="animate-fade-in space-y-4 md:space-y-5">
          <div className="flex items-center justify-between gap-3">
            <SkeletonBar className="h-10 w-44 md:h-11 md:w-56" />
            <SkeletonBar className="h-11 w-36 rounded-control" />
          </div>
          <SkeletonBar className="h-44 rounded-hero md:h-52" />
          <div className="grid grid-cols-2 gap-3 md:gap-5">
            <SkeletonBar className="h-28 rounded-card md:h-36" />
            <SkeletonBar className="h-28 rounded-card md:h-36" />
          </div>
          <SkeletonBar className="h-64 rounded-card" />
        </div>
      )}
    </div>
  )
}

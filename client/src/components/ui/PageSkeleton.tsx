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
        <div className="animate-fade-in space-y-6">
          <SkeletonBar className="h-10 w-56" />
          <div className="grid gap-4 md:grid-cols-3">
            <SkeletonBar className="h-36 rounded-[24px]" />
            <SkeletonBar className="h-36 rounded-[24px]" />
            <SkeletonBar className="h-36 rounded-[24px]" />
          </div>
          <SkeletonBar className="h-72 rounded-[24px]" />
        </div>
      )}
    </div>
  )
}

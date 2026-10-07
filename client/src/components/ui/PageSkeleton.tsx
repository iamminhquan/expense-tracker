import { useEffect, useState } from 'react'
import { SkeletonBar } from './SkeletonBar'

type SkeletonShape = 'dashboard' | 'list' | 'cards'

const tile = 'rounded-tile border border-border bg-surface'

function Row() {
  return (
    <div className="flex items-center gap-3 px-4 py-3.5 md:px-6">
      <SkeletonBar className="size-10 shrink-0" />
      <div className="flex-1 space-y-2">
        <SkeletonBar className="h-3.5 w-2/5" />
        <SkeletonBar className="h-3 w-1/4" />
      </div>
      <SkeletonBar className="h-4 w-20" />
    </div>
  )
}

// Waits 150ms before showing anything, so a fast response never flashes a skeleton.
export function PageSkeleton({ label = 'Loading…', shape = 'cards' }: { label?: string; shape?: SkeletonShape }) {
  const [visible, setVisible] = useState(false)
  useEffect(() => {
    const timer = window.setTimeout(() => setVisible(true), 150)
    return () => window.clearTimeout(timer)
  }, [])

  return (
    <div role="status" aria-busy="true">
      <span className="sr-only">{label}</span>
      {visible && (
        <div className="animate-fade-in space-y-5">
          <div className="flex items-end justify-between gap-4">
            <SkeletonBar className="h-9 w-48" />
            <SkeletonBar className="h-11 w-36" />
          </div>
          {shape === 'dashboard' && (
            <div className="grid gap-3 md:grid-cols-12 md:gap-5">
              <div className={`${tile} space-y-4 p-6 md:col-span-7 md:row-span-2`}>
                <SkeletonBar className="h-3.5 w-32" />
                <SkeletonBar className="h-14 w-3/4" />
                <SkeletonBar className="h-3 w-full" />
              </div>
              <div className="grid grid-cols-2 gap-3 md:col-span-5 md:grid-cols-1 md:gap-5">
                <div className={`${tile} space-y-3 p-5`}>
                  <SkeletonBar className="h-3 w-16" />
                  <SkeletonBar className="h-8 w-28" />
                </div>
                <div className={`${tile} space-y-3 p-5`}>
                  <SkeletonBar className="h-3 w-16" />
                  <SkeletonBar className="h-8 w-28" />
                </div>
              </div>
              <div className={`${tile} h-72 md:col-span-5`} />
              <div className={`${tile} h-72 md:col-span-7`} />
            </div>
          )}
          {shape === 'list' && (
            <div className={`${tile} divide-y divide-border overflow-hidden`}>
              <Row />
              <Row />
              <Row />
              <Row />
              <Row />
            </div>
          )}
          {shape === 'cards' && (
            <div className="grid gap-4 md:grid-cols-2 md:gap-5">
              <div className={`${tile} h-64`} />
              <div className={`${tile} h-64`} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

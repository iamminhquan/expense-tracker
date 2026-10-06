import type { ReactNode } from 'react'

interface CardProps {
  title: string
  children: ReactNode
}

export function Card({ title, children }: CardProps) {
  return (
    <div className="rounded-[16px] border border-border-card bg-surface p-5">
      <p className="mb-3 text-[14px] font-semibold">{title}</p>
      {children}
    </div>
  )
}

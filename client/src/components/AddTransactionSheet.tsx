import { useState } from 'react'
import { useCategories } from '../hooks/useCategories'
import { AddTransactionForm } from './AddTransactionForm'
import { BottomSheet } from './BottomSheet'
import { InlineError } from './ui/InlineError'
import { SkeletonBar } from './ui/SkeletonBar'

interface AddTransactionSheetProps {
  open: boolean
  onClose: () => void
}

export function AddTransactionSheet({ open, onClose }: AddTransactionSheetProps) {
  const { data, error, refetch } = useCategories()
  const categories = data ? data.expenseCategories.concat(data.incomeCategories) : []
  // A new key per opening gives a fresh form; it stays mounted while the sheet slides out.
  const [opening, setOpening] = useState(0)
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setOpening((n) => n + 1)
  }

  return (
    <BottomSheet open={open} onClose={onClose} label="New transaction">
      <h2 className="heading mb-4 text-[22px] leading-7 text-ink">New transaction</h2>
      {data ? (
        <AddTransactionForm key={opening} categories={categories} onAdded={onClose} autoFocusAmount={open} />
      ) : error ? (
        <InlineError message="Could not load your categories." onRetry={() => void refetch()} />
      ) : (
        <div role="status" className="space-y-4 pb-4">
          <span className="sr-only">Loading categories…</span>
          <SkeletonBar className="h-12 w-full" />
          <SkeletonBar className="h-[72px] w-full rounded-panel" />
          <SkeletonBar className="h-11 w-3/4" />
        </div>
      )}
    </BottomSheet>
  )
}

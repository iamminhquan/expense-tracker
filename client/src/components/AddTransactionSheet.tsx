import { useEffect, useRef } from 'react'
import { X } from 'lucide-react'
import { useCategories } from '../hooks/useCategories'
import { iconButtonClass } from '../lib/formStyles'
import { AddTransactionForm } from './AddTransactionForm'
import { BottomSheet } from './BottomSheet'

interface AddTransactionSheetProps {
  open: boolean
  onClose: () => void
}

// Mobile's way in to the add form from any page; desktop keeps it inline on Transactions.
export function AddTransactionSheet({ open, onClose }: AddTransactionSheetProps) {
  const { data: categories } = useCategories()
  const amountRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open) return
    const frame = requestAnimationFrame(() => amountRef.current?.focus())
    return () => cancelAnimationFrame(frame)
  }, [open])

  const all = categories ? categories.expenseCategories.concat(categories.incomeCategories) : []
  return (
    <BottomSheet open={open} onClose={onClose} label="Add transaction">
      <div className="relative">
        <button type="button" onClick={onClose} aria-label="Close" className={`${iconButtonClass} absolute -top-1 -right-2 z-10`}>
          <X aria-hidden="true" />
        </button>
        <AddTransactionForm categories={all} onAdded={onClose} amountRef={amountRef} />
      </div>
    </BottomSheet>
  )
}

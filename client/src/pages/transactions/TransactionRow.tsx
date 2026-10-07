import { useEffect, useRef, useState } from 'react'
import { Ellipsis, Pencil, Trash2, TriangleAlert } from 'lucide-react'
import { useDeleteTransaction } from '../../hooks/useTransactions'
import { useLongPress } from '../../hooks/useLongPress'
import { useIsDesktop } from '../../hooks/useMediaQuery'
import { BottomSheet } from '../../components/BottomSheet'
import { Badge } from '../../components/ui/Badge'
import { CategoryAvatar } from '../../components/ui/CategoryAvatar'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { ApiError } from '../../lib/api/client'
import { buttonClass, iconButtonClass } from '../../lib/formStyles'
import { formatDateLong, formatDateShort, formatVNDSigned } from '../../lib/format'
import { useToast } from '../../lib/toast/ToastContext'
import type { Category, Transaction } from '../../lib/api/types'
import { EditTransactionForm } from './EditTransactionForm'

interface TransactionRowProps {
  transaction: Transaction
  /** Rows grouped under a day heading leave the date out; a list sorted by amount shows it. */
  showDate: boolean
  showYear: boolean
  categories: Category[]
}

export function TransactionRow({ transaction, showDate, showYear, categories }: TransactionRowProps) {
  const deleteTransaction = useDeleteTransaction()
  const toast = useToast()
  const isDesktop = useIsDesktop()
  const [editing, setEditing] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [sheetMode, setSheetMode] = useState<'actions' | 'edit'>('actions')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const editButtonRef = useRef<HTMLButtonElement>(null)
  const wasEditing = useRef(false)

  function openSheet() {
    setSheetMode('actions')
    setSheetOpen(true)
  }

  /* Long-press adds to the visible "…" button below; it never replaces it. The sheet and the
     confirm dialog render inside the row, so presses in them bubble here and must be ignored. */
  const longPress = useLongPress(() => {
    if (!sheetOpen && !confirmOpen) openSheet()
  })

  // The Edit button only exists again after the re-render, so focus it from an effect.
  useEffect(() => {
    if (wasEditing.current && !editing) editButtonRef.current?.focus()
    wasEditing.current = editing
  }, [editing])

  const rowName = transaction.description || transaction.categoryName
  const date = showYear ? formatDateLong(transaction.occurredOn) : formatDateShort(transaction.occurredOn)
  const signed = formatVNDSigned(transaction.type === 'expense' ? -transaction.amount : transaction.amount)
  const amountClass = transaction.type === 'income' ? 'text-income' : 'text-ink'
  const avatar = <CategoryAvatar name={transaction.categoryName} color={transaction.categoryColor} />

  function askDelete() {
    setDeleteError(null)
    setConfirmOpen(true)
  }

  async function onDelete() {
    try {
      await deleteTransaction.mutateAsync(transaction.id)
      setConfirmOpen(false)
      toast.success('Transaction deleted')
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : 'Could not delete the transaction.')
    }
  }

  const duplicateBadge = transaction.isDuplicate && (
    <Badge kind="warning" icon={<TriangleAlert aria-hidden="true" />}>
      Possible duplicate
    </Badge>
  )
  const subtitle = (
    <p className="flex min-w-0 gap-2 text-[13px] leading-[18px] text-ink-muted">
      {transaction.description && <span className="truncate">{transaction.categoryName}</span>}
      {showDate && !isDesktop && <span className="tabular shrink-0">{date}</span>}
    </p>
  )

  const confirm = (
    <ConfirmDialog
      open={confirmOpen}
      title="Delete this transaction?"
      onCancel={() => setConfirmOpen(false)}
      onConfirm={() => void onDelete()}
      pending={deleteTransaction.isPending}
      error={deleteError}
    >
      <span className="font-semibold text-ink">{rowName}</span>, <span className="tabular whitespace-nowrap">{signed}</span> on{' '}
      {formatDateLong(transaction.occurredOn)}. This can't be undone.
    </ConfirmDialog>
  )

  if (isDesktop) {
    if (editing) {
      return (
        <li className="animate-fade-in bg-surface-2">
          <EditTransactionForm transaction={transaction} categories={categories} layout="inline" onDone={() => setEditing(false)} />
        </li>
      )
    }
    return (
      <li className="group flex min-h-16 items-center gap-4 px-5 py-2.5 hover:bg-[color-mix(in_srgb,var(--color-surface),var(--color-ink)_3%)]">
        {avatar}
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</p>
            {duplicateBadge}
          </div>
          {subtitle}
        </div>
        {showDate && <span className="tabular w-[104px] shrink-0 text-[14px] leading-5 text-ink-muted">{date}</span>}
        <span className={`figure-sm w-[150px] shrink-0 text-right text-[18px] leading-6 whitespace-nowrap ${amountClass}`}>{signed}</span>
        <div className="flex shrink-0 gap-0.5">
          <button
            ref={editButtonRef}
            type="button"
            onClick={() => setEditing(true)}
            aria-label={`Edit ${rowName}`}
            className={buttonClass('ghost', 'sm')}
          >
            <Pencil aria-hidden="true" />
            Edit
          </button>
          <button type="button" onClick={askDelete} aria-label={`Delete ${rowName}`} className={buttonClass('danger-ghost', 'sm')}>
            <Trash2 aria-hidden="true" />
            Delete
          </button>
        </div>
        {confirm}
      </li>
    )
  }

  return (
    <li className="flex min-h-[68px] items-center gap-3 py-2.5 pr-1.5 pl-4 select-none [-webkit-touch-callout:none] active:bg-surface-2" {...longPress}>
      {avatar}
      <div className="min-w-0 flex-1">
        <p className="truncate text-[15px] leading-[22px] font-semibold text-ink">{rowName}</p>
        {subtitle}
        {duplicateBadge && <div className="mt-1">{duplicateBadge}</div>}
      </div>
      <span className={`figure-sm text-[17px] leading-6 whitespace-nowrap ${amountClass}`}>{signed}</span>
      <button type="button" onClick={openSheet} aria-haspopup="dialog" aria-label={`Actions for ${rowName}`} className={iconButtonClass}>
        <Ellipsis aria-hidden="true" />
      </button>

      <BottomSheet open={sheetOpen} onClose={() => setSheetOpen(false)} label={sheetMode === 'edit' ? `Edit ${rowName}` : `Actions for ${rowName}`}>
        {sheetMode === 'actions' ? (
          <>
            <div className="flex items-center gap-3">
              <CategoryAvatar name={transaction.categoryName} color={transaction.categoryColor} size="lg" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-[16px] leading-6 font-semibold text-ink">{rowName}</p>
                <p className="truncate text-[13px] leading-[18px] text-ink-muted">
                  {transaction.categoryName}, <span className="tabular">{formatDateLong(transaction.occurredOn)}</span>
                </p>
              </div>
            </div>
            <p className={`figure mt-5 text-[44px] leading-none ${amountClass}`}>{signed}</p>
            {duplicateBadge && <div className="mt-3">{duplicateBadge}</div>}
            <div className="mt-6 grid grid-cols-2 gap-3">
              <button type="button" onClick={() => setSheetMode('edit')} className={buttonClass('secondary', 'lg')}>
                <Pencil aria-hidden="true" />
                Edit
              </button>
              <button
                type="button"
                onClick={() => {
                  setSheetOpen(false)
                  askDelete()
                }}
                className={buttonClass('danger-tint', 'lg')}
              >
                <Trash2 aria-hidden="true" />
                Delete
              </button>
            </div>
          </>
        ) : (
          <>
            <h2 className="heading mb-4 text-[22px] leading-7 text-ink">Edit transaction</h2>
            <EditTransactionForm transaction={transaction} categories={categories} layout="stacked" onDone={() => setSheetOpen(false)} />
          </>
        )}
      </BottomSheet>
      {confirm}
    </li>
  )
}

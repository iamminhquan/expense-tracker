import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useDeleteAccount } from '../../hooks/useSettings'
import { useAuth } from '../../lib/auth/AuthContext'
import { ApiError } from '../../lib/api/client'
import { downloadTransactionsExport } from '../../lib/api/import'
import { inputClass } from '../../lib/formStyles'

export function DangerZoneCard() {
  const deleteAccount = useDeleteAccount()
  const { logout } = useAuth()
  const navigate = useNavigate()
  const [currentPassword, setCurrentPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

  async function onDelete(e: FormEvent) {
    e.preventDefault()
    if (!confirm('Delete your account? This cannot be undone.')) return
    setError(null)
    try {
      await deleteAccount.mutateAsync(currentPassword)
      await logout()
      navigate('/login', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete account.')
    }
  }

  async function onExport() {
    setExportError(null)
    setExporting(true)
    try {
      await downloadTransactionsExport('?month=all')
    } catch (err) {
      setExportError(err instanceof ApiError ? err.message : 'Could not export transactions.')
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="rounded-[16px] border border-danger-border bg-danger-tint p-5">
      <p className="mb-3 text-[14px] font-semibold text-expense">Danger zone</p>
      <p className="mb-2 text-[13px] text-ink-muted">Deleting your account removes every transaction. Keep a copy first.</p>
      <button
        type="button"
        onClick={() => void onExport()}
        disabled={exporting}
        className="mb-1 text-[13px] text-accent hover:underline disabled:opacity-60"
      >
        {exporting ? 'Exporting…' : 'Export all transactions (CSV)'}
      </button>
      {exportError && (
        <p role="alert" className="mb-2 text-[13px] text-expense">
          {exportError}
        </p>
      )}
      <hr className="my-4 border-danger-border" />
      <form onSubmit={onDelete} className="space-y-3">
        <input
          type="password"
          className={inputClass}
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          placeholder="Current password"
          required
        />
        {error && <p className="text-[13px] text-expense">{error}</p>}
        <button
          type="submit"
          disabled={deleteAccount.isPending}
          className="w-full rounded-[10px] bg-expense px-4 py-2.5 text-[14px] font-semibold text-on-solid hover:opacity-90 disabled:opacity-50"
        >
          Delete account
        </button>
      </form>
    </div>
  )
}

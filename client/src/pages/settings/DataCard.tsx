import { useState } from 'react'
import { Download, Upload } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ApiError } from '../../lib/api/client'
import { downloadTransactionsExport } from '../../lib/api/import'
import { buttonClass } from '../../lib/formStyles'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { Card } from './Card'

export function DataCard() {
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

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
    <Card title="Your data" description="Bring in a statement from your bank, or take every transaction out as a CSV that opens in any spreadsheet.">
      <div>
        <div className="grid gap-3 sm:flex sm:flex-wrap">
          <Link to="/transactions/import" className={buttonClass('secondary')}>
            <Upload aria-hidden="true" />
            Import CSV
          </Link>
          <button type="button" onClick={() => void onExport()} disabled={exporting} aria-busy={exporting} className={buttonClass('secondary')}>
            <Download aria-hidden="true" />
            Export all transactions (CSV)
          </button>
        </div>
        {exportError && (
          <div role="alert">
            <FieldErrorText>{exportError}</FieldErrorText>
          </div>
        )}
      </div>
    </Card>
  )
}

import { getAccessToken } from './tokenStore'
import { ApiError } from './client'

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''

export interface MappingGuess {
  dateCol: number
  amountCol: number
  typeCol: number
  categoryCol: number
  noteCol: number
  dateLayout: string
  negativeIsExpense: boolean
  fallbackCategory: string
}

export interface DateFormat {
  key: string
  label: string
}

export interface MappingNeeded {
  needsMapping: true
  columns: string[]
  sample: string[][]
  rows: number
  guess: MappingGuess
  dateFormats: DateFormat[]
  ambiguousDate: boolean
  categoryNames: string[]
  fingerprint: string
}

export interface ImportPreview {
  preview: true
  rowCount: number
  newCategories: { name: string; type: string }[]
  errors: { line: number; message: string }[]
  moreErrors: number
  rounded: number
  duplicates: number
  fingerprint: string
  importable: boolean
  dateSuspect: boolean
}

export interface ImportResult {
  imported: number
  month: string
}

export interface MappingFields {
  dateCol: number
  amountCol: number
  typeCol: number
  categoryCol: number
  noteCol: number
  dateLayout: string
  negativeIsExpense: boolean
  fallbackCategory: string
}

// importTransactions is a one-off multipart POST rather than going through
// client.ts's api.post: that helper always sends Content-Type:
// application/json, which a file upload can't use. The access token and
// 401-retry-via-refresh logic are reused by hand here instead of pulled
// into client.ts's request(), since this is the only caller that needs
// multipart at all.
export async function importTransactions(
  file: File,
  options: { mapping?: MappingFields; confirm?: boolean; fingerprint?: string } = {},
): Promise<MappingNeeded | ImportPreview | ImportResult> {
  const form = new FormData()
  form.set('file', file)
  if (options.mapping) {
    form.set('mapped', '1')
    form.set('date_col', String(options.mapping.dateCol))
    form.set('amount_col', String(options.mapping.amountCol))
    form.set('type_col', String(options.mapping.typeCol))
    form.set('category_col', String(options.mapping.categoryCol))
    form.set('note_col', String(options.mapping.noteCol))
    form.set('date_layout', options.mapping.dateLayout)
    if (options.mapping.negativeIsExpense) form.set('negative_is_expense', '1')
    form.set('fallback_category', options.mapping.fallbackCategory)
  }
  if (options.confirm) {
    form.set('confirm', '1')
    form.set('fingerprint', options.fingerprint ?? '')
  }

  const token = getAccessToken()
  const res = await fetch(`${API_BASE}/api/transactions/import`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    credentials: 'include',
    body: form,
  })
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string }
    throw new ApiError(res.status, data.error ?? res.statusText)
  }
  return res.json()
}

// downloadTransactionsExport fetches the CSV with the access token
// attached (a plain <a href> can't carry an Authorization header) and
// turns the response into a browser download via an object URL -- the
// standard pattern for an authenticated file download from a SPA.
export async function downloadTransactionsExport(query: string): Promise<void> {
  const token = getAccessToken()
  const res = await fetch(`${API_BASE}/api/transactions/export${query}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    credentials: 'include',
  })
  if (!res.ok) {
    throw new ApiError(res.status, 'Could not export transactions.')
  }
  const blob = await res.blob()
  const filename = res.headers.get('Content-Disposition')?.match(/filename="(.+)"/)?.[1] ?? 'spend-export.csv'
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

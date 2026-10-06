import { ApiError, authedFetch, readApiResponse } from './client'

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

// Multipart can't use api.post (always JSON), hence authedFetch.
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

  const res = await authedFetch('/api/v1/transactions/import', { method: 'POST', body: form })
  return readApiResponse(res)
}

// <a href> can't send the Authorization header, so fetch it and save via an object URL.
export async function downloadTransactionsExport(query: string): Promise<void> {
  const res = await authedFetch(`/api/v1/transactions/export${query}`)
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

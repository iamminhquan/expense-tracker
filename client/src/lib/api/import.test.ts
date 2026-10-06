import { beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadTransactionsExport, importTransactions, type MappingFields } from './import'
import { setAccessToken, setUnauthorizedHandler } from './tokenStore'

const mapping: MappingFields = {
  dateCol: 0,
  amountCol: 2,
  typeCol: -1,
  categoryCol: 3,
  noteCol: 1,
  dateLayout: '02/01/2006',
  negativeIsExpense: true,
  fallbackCategory: 'Other',
}

const csv = () => new File(['date,amount\n2026-01-02,5000\n'], 'bank.csv', { type: 'text/csv' })

function envelope(data: unknown, status = 200, success = true, message = 'ok'): Response {
  return new Response(JSON.stringify({ success, message, data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

const urlOf = (input: RequestInfo | URL) => (typeof input === 'string' ? input : input.toString())
const authOf = (init?: RequestInit) => new Headers(init?.headers).get('Authorization')

beforeEach(() => {
  setAccessToken('tok')
  setUnauthorizedHandler(null)
})

describe('importTransactions', () => {
  it('sends just the file for the first look at it', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ needsMapping: true }))
    vi.stubGlobal('fetch', fetchMock)

    await importTransactions(csv())

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/transactions/import')
    expect(init.method).toBe('POST')
    expect(authOf(init)).toBe('Bearer tok')
    const form = init.body as FormData
    expect([...form.keys()]).toEqual(['file'])
    expect((form.get('file') as File).name).toBe('bank.csv')
  })

  it('sends the mapping the way the server reads it', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ preview: true }))
    vi.stubGlobal('fetch', fetchMock)

    await importTransactions(csv(), { mapping })

    const form = (fetchMock.mock.calls[0][1] as RequestInit).body as FormData
    expect(Object.fromEntries([...form.entries()].filter(([k]) => k !== 'file'))).toEqual({
      mapped: '1',
      date_col: '0',
      amount_col: '2',
      type_col: '-1',
      category_col: '3',
      note_col: '1',
      date_layout: '02/01/2006',
      negative_is_expense: '1',
      fallback_category: 'Other',
    })
  })

  it('leaves negative_is_expense out when it is off, since any value turns it on', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ preview: true }))
    vi.stubGlobal('fetch', fetchMock)

    await importTransactions(csv(), { mapping: { ...mapping, negativeIsExpense: false } })

    expect(((fetchMock.mock.calls[0][1] as RequestInit).body as FormData).has('negative_is_expense')).toBe(false)
  })

  it('confirms with the fingerprint of the preview the user saw', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope({ imported: 3, month: '2026-01' }))
    vi.stubGlobal('fetch', fetchMock)

    const result = await importTransactions(csv(), { mapping, confirm: true, fingerprint: 'abc123' })

    const form = (fetchMock.mock.calls[0][1] as RequestInit).body as FormData
    expect(form.get('confirm')).toBe('1')
    expect(form.get('fingerprint')).toBe('abc123')
    expect(result).toEqual({ imported: 3, month: '2026-01' })
  })

  it('throws the server message when the file is rejected', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(envelope(null, 400, false, 'choose a .csv file smaller than 1 MB')))

    await expect(importTransactions(csv())).rejects.toMatchObject({
      status: 400,
      message: 'choose a .csv file smaller than 1 MB',
    })
  })

  // The access token lives 15 minutes; a person can sit on the mapping screen
  // longer than that, and the confirm must not fail for it.
  it('refreshes an expired access token and retries the upload', async () => {
    setAccessToken('expired')
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (urlOf(input) === '/api/v1/refresh') return envelope({ accessToken: 'fresh' })
      return authOf(init) === 'Bearer fresh' ? envelope({ imported: 1, month: '2026-01' }) : envelope(null, 401, false, 'expired')
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(importTransactions(csv(), { mapping, confirm: true, fingerprint: 'f' })).resolves.toEqual({
      imported: 1,
      month: '2026-01',
    })
    const retried = fetchMock.mock.calls.at(-1) as [string, RequestInit]
    expect(((retried[1].body as FormData).get('file') as File).name).toBe('bank.csv')
  })
})

describe('downloadTransactionsExport', () => {
  it('saves the response under the server-chosen filename', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('date,amount\n', { headers: { 'Content-Disposition': 'attachment; filename="spend-2026-01.csv"' } }),
      ),
    )
    URL.createObjectURL = vi.fn(() => 'blob:x')
    URL.revokeObjectURL = vi.fn()
    let saved: string | undefined
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      saved = this.download
    })

    await downloadTransactionsExport('?month=2026-01')

    expect(saved).toBe('spend-2026-01.csv')
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:x')
    expect(document.querySelector('a[download]')).toBeNull()
  })

  it('refreshes an expired access token and retries', async () => {
    setAccessToken('expired')
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (urlOf(input) === '/api/v1/refresh') return envelope({ accessToken: 'fresh' })
      return authOf(init) === 'Bearer fresh' ? new Response('x') : envelope(null, 401, false, 'expired')
    })
    vi.stubGlobal('fetch', fetchMock)
    URL.createObjectURL = vi.fn(() => 'blob:x')
    URL.revokeObjectURL = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await expect(downloadTransactionsExport('')).resolves.toBeUndefined()
  })

  it('reports a failed export instead of saving an error page', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(envelope(null, 500, false, 'boom')))
    const createObjectURL = vi.fn()
    URL.createObjectURL = createObjectURL

    await expect(downloadTransactionsExport('')).rejects.toMatchObject({ status: 500 })
    expect(createObjectURL).not.toHaveBeenCalled()
  })
})

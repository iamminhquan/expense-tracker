import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setAccessToken } from '../../lib/api/tokenStore'
import { ImportPage } from './ImportPage'

const guess = {
  dateCol: 0,
  amountCol: 2,
  typeCol: -1,
  categoryCol: -1,
  noteCol: 1,
  dateLayout: '02/01/2006',
  negativeIsExpense: true,
  fallbackCategory: 'Other',
}

const needsMapping = {
  needsMapping: true,
  columns: ['Date', 'Memo', 'Amount'],
  sample: [['02/01/2026', 'Coffee', '-45000']],
  rows: 2,
  guess,
  dateFormats: [{ key: '02/01/2006', label: 'Day first (31/12/2026)' }],
  ambiguousDate: false,
  categoryNames: ['Food', 'Other'],
  fingerprint: 'fp-1',
}

const preview = {
  preview: true,
  rowCount: 2,
  newCategories: [{ name: 'Coffee', type: 'expense' }],
  errors: [],
  moreErrors: 0,
  rounded: 0,
  duplicates: 0,
  fingerprint: 'fp-2',
  importable: true,
  dateSuspect: false,
}

function envelope(data: unknown, status = 200, success = true, message = 'ok'): Response {
  return new Response(JSON.stringify({ success, message, data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function Where() {
  const { pathname, search } = useLocation()
  return <p data-testid="where">{pathname + search}</p>
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/import']}>
        <Routes>
          <Route path="/import" element={<ImportPage />} />
          <Route path="/transactions" element={<Where />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return { invalidate }
}

/** The multipart fields of each import request, in order. */
function sentForms(fetchMock: ReturnType<typeof vi.fn>): Record<string, string>[] {
  return fetchMock.mock.calls.map(([, init]) =>
    Object.fromEntries([...((init as RequestInit).body as FormData).entries()].map(([k, v]) => [k, typeof v === 'string' ? v : v.name])),
  )
}

const csv = new File(['Date,Memo,Amount\n02/01/2026,Coffee,-45000\n'], 'bank.csv', { type: 'text/csv' })

beforeEach(() => setAccessToken('tok'))

describe('ImportPage', () => {
  it('walks a file the server cannot read on its own through mapping, preview and confirm', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(envelope(needsMapping))
      .mockResolvedValueOnce(envelope(preview))
      .mockResolvedValueOnce(envelope({ imported: 2, month: '2026-01' }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const { invalidate } = renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)
    expect(await screen.findByText("We don't recognize this format")).toBeTruthy()

    await user.click(screen.getByRole('button', { name: 'Preview import' }))
    await user.click(await screen.findByRole('button', { name: 'Import 2 rows' }))
    expect(await screen.findByText('Imported 2 transactions.')).toBeTruthy()

    const [first, second, third] = sentForms(fetchMock)
    expect(first).toEqual({ file: 'bank.csv' })
    expect(second).toMatchObject({ file: 'bank.csv', mapped: '1', date_col: '0', amount_col: '2', negative_is_expense: '1' })
    // Confirming repeats the mapping and names the preview that was shown.
    expect(third).toMatchObject({ mapped: '1', date_col: '0', confirm: '1', fingerprint: 'fp-2' })

    // What the import changed must not be served from cache.
    const invalidated = invalidate.mock.calls.map(([f]) => (f as { queryKey: string[] }).queryKey[0])
    expect(invalidated).toEqual(expect.arrayContaining(['transactions', 'dashboard', 'categories']))

    await user.click(screen.getByRole('button', { name: 'View transactions' }))
    expect((await screen.findByTestId('where')).textContent).toBe('/transactions?month=2026-01')
  })

  it('sends the mapping the person changed, not the server guess', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(envelope(needsMapping)).mockResolvedValueOnce(envelope(preview))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)
    await screen.findByText("We don't recognize this format")
    const [, amount] = screen.getAllByRole('combobox')
    await user.selectOptions(amount, 'Memo')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Preview import' }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    const second = sentForms(fetchMock)[1]
    expect(second.amount_col).toBe('1')
    expect(second.negative_is_expense).toBeUndefined()
  })

  it('goes straight to the preview for a file the server recognizes', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(envelope(preview))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)

    expect(await screen.findByRole('button', { name: 'Import 2 rows' })).toBeTruthy()
    expect(screen.queryByText("We don't recognize this format")).toBeNull()
  })

  it('shows the server reason when a file is rejected, and stays on the upload step', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(envelope(null, 400, false, 'choose a .csv file smaller than 1 MB')))
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)

    expect(await screen.findByText('choose a .csv file smaller than 1 MB')).toBeTruthy()
    expect(document.querySelector('input[type=file]')).not.toBeNull()
  })

  it('will not import a preview the server marked not importable, and lists what is wrong', async () => {
    const blocked = {
      ...preview,
      importable: false,
      rowCount: 0,
      errors: [{ line: 3, message: 'amount is not a number' }],
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(envelope(blocked)))
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)

    const button = await screen.findByRole('button', { name: 'Import 0 rows' })
    expect((button as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText('Line 3: amount is not a number')).toBeTruthy()
  })

  it('shows a failed confirm and lets the person try again', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(envelope(preview))
      .mockResolvedValueOnce(envelope(null, 409, false, 'the file changed since the preview'))
      .mockResolvedValueOnce(envelope({ imported: 1, month: '2026-01' }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)
    await user.click(await screen.findByRole('button', { name: 'Import 2 rows' }))
    expect(await screen.findByText('the file changed since the preview')).toBeTruthy()

    await user.click(screen.getByRole('button', { name: 'Import 2 rows' }))
    expect(await screen.findByText('Imported 1 transaction.')).toBeTruthy()
  })

  it('goes back to the upload step on "Start over"', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(envelope(needsMapping)))
    const user = userEvent.setup()
    renderPage()

    await user.upload(document.querySelector('input[type=file]')!, csv)
    const form = (await screen.findByText("We don't recognize this format")).closest('div')!
    await user.click(within(form).getByRole('button', { name: 'Start over' }))

    expect(document.querySelector('input[type=file]')).not.toBeNull()
  })
})

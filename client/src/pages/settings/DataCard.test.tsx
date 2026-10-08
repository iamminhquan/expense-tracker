import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setAccessToken } from '../../lib/api/tokenStore'
import { DataCard } from './DataCard'

function renderCard() {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <DataCard />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  setAccessToken('tok')
  URL.createObjectURL = vi.fn(() => 'blob:x')
  URL.revokeObjectURL = vi.fn()
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

describe('DataCard export', () => {
  // The export defaults to the current month; leaving out "all" would hand over one month's worth.
  it('exports every month, not just the current one', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('Date,Type\n'))
    vi.stubGlobal('fetch', fetchMock)
    renderCard()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Export all transactions (CSV)' }))

    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/transactions/export?month=all')
    expect(await screen.findByRole('button', { name: 'Export all transactions (CSV)' })).toBeTruthy()
  })

  it('says so when the export fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 500 })))
    renderCard()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Export all transactions (CSV)' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Could not export transactions.')
  })

  it('links to the CSV import', () => {
    renderCard()
    expect(screen.getByRole('link', { name: 'Import CSV' }).getAttribute('href')).toBe('/transactions/import')
  })
})

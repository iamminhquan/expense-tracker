import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { User } from '../../lib/api/types'
import { setAccessToken } from '../../lib/api/tokenStore'
import { VerifyEmailBanner } from './VerifyEmailBanner'

let user: User | null

vi.mock('../../lib/auth/AuthContext', () => ({ useAuth: () => ({ user }) }))

const baseUser: User = { id: 1, name: 'Demo', email: 'demo@example.com', username: 'demo', theme: 'auto', emailVerified: false }

function envelope(data: unknown, status = 200, success = true, message = 'ok'): Response {
  return new Response(JSON.stringify({ success, message, data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function renderBanner() {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { mutations: { retry: false } } })}>
      <VerifyEmailBanner />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  user = baseUser
  setAccessToken('tok')
})

describe('VerifyEmailBanner', () => {
  it('asks an unconfirmed account to confirm, naming the address', () => {
    renderBanner()
    expect(screen.getByRole('region', { name: 'Email confirmation' }).textContent).toContain('demo@example.com')
  })

  it('shows nothing once the email is confirmed', () => {
    user = { ...baseUser, emailVerified: true }
    renderBanner()
    expect(screen.queryByRole('region', { name: 'Email confirmation' })).toBeNull()
  })

  it('shows nothing when signed out', () => {
    user = null
    renderBanner()
    expect(screen.queryByRole('region', { name: 'Email confirmation' })).toBeNull()
  })

  it('resends the link and says so', async () => {
    const fetchMock = vi.fn().mockResolvedValue(envelope(null))
    vi.stubGlobal('fetch', fetchMock)
    renderBanner()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Resend link' }))

    expect(await screen.findByText('Link sent. Check your inbox.')).toBeTruthy()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/settings/resend-verification')
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe('POST')
  })

  it('shows why a resend failed, then can try again', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(envelope(null, 429, false, 'too many requests, try again later'))
      .mockResolvedValueOnce(envelope(null))
    vi.stubGlobal('fetch', fetchMock)
    renderBanner()
    const click = () => userEvent.setup().click(screen.getByRole('button', { name: 'Resend link' }))

    await click()
    expect(await screen.findByText('too many requests, try again later')).toBeTruthy()

    await click()
    expect(await screen.findByText('Link sent. Check your inbox.')).toBeTruthy()
    expect(screen.queryByText('too many requests, try again later')).toBeNull()
  })
})

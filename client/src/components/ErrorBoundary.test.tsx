import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ErrorBoundary } from './ErrorBoundary'

let failure: Error | null

function Page() {
  if (failure) throw failure
  return <p>All good</p>
}

beforeEach(() => {
  failure = null
  // React logs every caught render error; keep the test output readable.
  vi.spyOn(console, 'error').mockImplementation(() => {})
})

describe('ErrorBoundary', () => {
  it('renders its children when nothing fails', () => {
    render(
      <ErrorBoundary>
        <Page />
      </ErrorBoundary>,
    )
    expect(screen.getByText('All good')).toBeTruthy()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('replaces a crashed page with an alert instead of a blank screen', () => {
    failure = new Error('cannot read properties of undefined')
    render(
      <ErrorBoundary>
        <Page />
      </ErrorBoundary>,
    )
    const alert = screen.getByRole('alert')
    expect(alert.textContent).toContain('Something went wrong')
    expect(alert.textContent).not.toContain('cannot read properties')
    expect(screen.queryByText('All good')).toBeNull()
  })

  it('"Try again" shows the page again once what broke is gone', async () => {
    failure = new Error('boom')
    render(
      <ErrorBoundary>
        <Page />
      </ErrorBoundary>,
    )
    failure = null
    await userEvent.setup().click(screen.getByRole('button', { name: 'Try again' }))
    expect(screen.getByText('All good')).toBeTruthy()
  })

  it('recovers by itself when resetKey changes, as it does on leaving the broken route', async () => {
    function Harness() {
      const [path, setPath] = useState('/broken')
      return (
        <>
          <button onClick={() => setPath('/fine')}>go</button>
          <ErrorBoundary resetKey={path}>{path === '/broken' ? <Page /> : <p>Other page</p>}</ErrorBoundary>
        </>
      )
    }
    failure = new Error('boom')
    render(<Harness />)
    expect(screen.getByRole('alert')).toBeTruthy()

    await userEvent.setup().click(screen.getByRole('button', { name: 'go' }))
    expect(screen.getByText('Other page')).toBeTruthy()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('stays on the error while resetKey is unchanged', () => {
    failure = new Error('boom')
    const { rerender } = render(
      <ErrorBoundary resetKey="/a">
        <Page />
      </ErrorBoundary>,
    )
    rerender(
      <ErrorBoundary resetKey="/a">
        <Page />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('alert')).toBeTruthy()
  })

  // A deploy replaces the hashed chunks, so a tab opened before it fails on the next lazy route.
  it('tells a stale tab to reload rather than to try again', () => {
    failure = new TypeError('Failed to fetch dynamically imported module: https://x/assets/DashboardPage-abc.js')
    render(
      <ErrorBoundary>
        <Page />
      </ErrorBoundary>,
    )
    expect(screen.getByRole('alert').textContent).toContain('has been updated')
    expect(screen.queryByRole('button', { name: 'Try again' })).toBeNull()
  })

  it('reloads the page on "Reload page"', async () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { ...window.location, reload })
    failure = new Error('boom')
    render(
      <ErrorBoundary>
        <Page />
      </ErrorBoundary>,
    )
    await userEvent.setup().click(screen.getByRole('button', { name: 'Reload page' }))
    expect(reload).toHaveBeenCalledTimes(1)
  })
})

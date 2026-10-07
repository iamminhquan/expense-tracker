import { Component, type ErrorInfo, type ReactNode } from 'react'
import { buttonClass } from '../lib/formStyles'

interface ErrorBoundaryProps {
  children: ReactNode
  /** The boundary clears itself when this changes, e.g. the pathname, so leaving a broken page recovers. */
  resetKey?: unknown
  /** Fill the screen instead of the space the children had, for a boundary around the whole app. */
  fullPage?: boolean
}

interface ErrorBoundaryState {
  error: Error | null
  resetKey: unknown
}

// What a browser throws when a lazy route's chunk is gone, which is what a deploy does to an open tab.
const CHUNK_LOAD_ERROR = /dynamically imported module|importing a module script failed|loading chunk/i

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null, resetKey: this.props.resetKey }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { error }
  }

  static getDerivedStateFromProps(props: ErrorBoundaryProps, state: ErrorBoundaryState): Partial<ErrorBoundaryState> | null {
    return props.resetKey === state.resetKey ? null : { error: null, resetKey: props.resetKey }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(error, info.componentStack)
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children

    const stale = CHUNK_LOAD_ERROR.test(error.message)
    return (
      <div
        role="alert"
        className={`flex flex-col items-center justify-center gap-4 px-4 text-center ${
          this.props.fullPage ? 'min-h-screen bg-app text-ink' : 'min-h-[300px]'
        }`}
      >
        <h1 className="heading text-[24px] leading-8 text-ink">{stale ? '$pend has been updated' : 'Something went wrong'}</h1>
        <p className="max-w-[420px] text-[15px] leading-[22px] text-ink-muted">
          {stale
            ? 'Reload to get the latest version.'
            : 'An unexpected error stopped this page. Your data is safe. Try again, or reload if it keeps happening.'}
        </p>
        <div className="flex gap-2">
          {!stale && (
            <button
              type="button"
              onClick={() => this.setState({ error: null })}
              className={buttonClass('secondary')}
            >
              Try again
            </button>
          )}
          <button
            type="button"
            onClick={() => window.location.reload()}
            className={buttonClass('primary')}
          >
            Reload page
          </button>
        </div>
      </div>
    )
  }
}

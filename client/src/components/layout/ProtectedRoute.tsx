import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { LoaderCircle } from 'lucide-react'
import { useAuth } from '../../lib/auth/AuthContext'

// Render nothing until the silent refresh settles, or each reload flashes the login page.
export function ProtectedRoute() {
  const { status } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return (
      <div role="status" className="flex min-h-screen items-center justify-center bg-app">
        <LoaderCircle aria-hidden="true" className="size-7 animate-spin text-ink-muted motion-reduce:animate-none" />
        <span className="sr-only">Loading…</span>
      </div>
    )
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <Outlet />
}

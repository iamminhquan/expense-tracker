import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../../lib/auth/AuthContext'

// Gates every /dashboard, /transactions, /categories, /settings route.
// "loading" (the bootstrap silent-refresh in AuthProvider hasn't settled
// yet) renders nothing rather than redirecting -- redirecting first and
// then finding out the refresh succeeded would flash the login page on
// every reload.
export function ProtectedRoute() {
  const { status } = useAuth()
  const location = useLocation()

  if (status === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-app">
        <p className="text-ink-faint">Loading…</p>
      </div>
    )
  }

  if (status === 'unauthenticated') {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <Outlet />
}

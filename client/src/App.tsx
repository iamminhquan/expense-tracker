import { lazy, Suspense } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { Navigate, BrowserRouter, Route, Routes } from 'react-router-dom'
import { queryClient } from './lib/queryClient'
import { AuthProvider } from './lib/auth/AuthContext'
import { ThemeProvider } from './lib/theme/ThemeContext'
import { ProtectedRoute } from './components/layout/ProtectedRoute'
import { Layout } from './components/layout/Layout'

// Every page is its own chunk, fetched the first time its route is
// visited rather than up front. The auth pages (reached by a signed-out
// visitor's very first paint) are the ones most worth not bundling
// together with Chart.js-heavy Dashboard or the rest of the authenticated
// app a first-time visitor hasn't even logged into yet.
const AuthPage = lazy(() => import('./pages/auth/AuthPage').then((m) => ({ default: m.AuthPage })))
const ForgotPasswordPage = lazy(() => import('./pages/auth/ForgotPasswordPage').then((m) => ({ default: m.ForgotPasswordPage })))
const ResetPasswordPage = lazy(() => import('./pages/auth/ResetPasswordPage').then((m) => ({ default: m.ResetPasswordPage })))
const VerifyEmailPage = lazy(() => import('./pages/auth/VerifyEmailPage').then((m) => ({ default: m.VerifyEmailPage })))
const DashboardPage = lazy(() => import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const TransactionsPage = lazy(() => import('./pages/TransactionsPage').then((m) => ({ default: m.TransactionsPage })))
const ImportPage = lazy(() => import('./pages/ImportPage').then((m) => ({ default: m.ImportPage })))
const CategoriesPage = lazy(() => import('./pages/CategoriesPage').then((m) => ({ default: m.CategoriesPage })))
const SettingsPage = lazy(() => import('./pages/SettingsPage').then((m) => ({ default: m.SettingsPage })))

// RouteFallback is deliberately minimal (no spinner graphic, just text):
// a route chunk is small and already cached after the first visit, so
// this is on screen for a blink at most, the same budget
// ProtectedRoute's own "Loading…" state gets for the auth bootstrap.
function RouteFallback() {
  return (
    <div className="flex min-h-[200px] items-center justify-center">
      <p className="text-ink-faint">Loading…</p>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <ThemeProvider>
            <Suspense fallback={<RouteFallback />}>
              <Routes>
                <Route path="/login" element={<AuthPage tab="login" />} />
                <Route path="/register" element={<AuthPage tab="register" />} />
                <Route path="/forgot-password" element={<ForgotPasswordPage />} />
                <Route path="/reset-password" element={<ResetPasswordPage />} />
                <Route path="/verify-email" element={<VerifyEmailPage />} />

                <Route element={<ProtectedRoute />}>
                  <Route element={<Layout />}>
                    <Route path="/" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/dashboard" element={<DashboardPage />} />
                    <Route path="/transactions" element={<TransactionsPage />} />
                    <Route path="/transactions/import" element={<ImportPage />} />
                    <Route path="/categories" element={<CategoriesPage />} />
                    <Route path="/settings" element={<SettingsPage />} />
                  </Route>
                </Route>

                <Route path="*" element={<Navigate to="/dashboard" replace />} />
              </Routes>
            </Suspense>
          </ThemeProvider>
        </AuthProvider>
      </BrowserRouter>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}

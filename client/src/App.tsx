import { lazy, Suspense } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { Navigate, BrowserRouter, Route, Routes } from 'react-router-dom'
import { queryClient } from './lib/queryClient'
import { AuthProvider } from './lib/auth/AuthContext'
import { ThemeProvider } from './lib/theme/ThemeContext'
import { ToastProvider } from './lib/toast/ToastContext'
import { ProtectedRoute } from './components/layout/ProtectedRoute'
import { Layout } from './components/layout/Layout'

// One chunk per route, so a signed-out visitor never downloads Chart.js.
const AuthPage = lazy(() => import('./pages/auth/AuthPage').then((m) => ({ default: m.AuthPage })))
const ForgotPasswordPage = lazy(() => import('./pages/auth/ForgotPasswordPage').then((m) => ({ default: m.ForgotPasswordPage })))
const ResetPasswordPage = lazy(() => import('./pages/auth/ResetPasswordPage').then((m) => ({ default: m.ResetPasswordPage })))
const VerifyEmailPage = lazy(() => import('./pages/auth/VerifyEmailPage').then((m) => ({ default: m.VerifyEmailPage })))
const DashboardPage = lazy(() => import('./pages/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const TransactionsPage = lazy(() => import('./pages/transactions/TransactionsPage').then((m) => ({ default: m.TransactionsPage })))
const ImportPage = lazy(() => import('./pages/import/ImportPage').then((m) => ({ default: m.ImportPage })))
const CategoriesPage = lazy(() => import('./pages/categories/CategoriesPage').then((m) => ({ default: m.CategoriesPage })))
const SettingsPage = lazy(() => import('./pages/settings/SettingsPage').then((m) => ({ default: m.SettingsPage })))

function RouteFallback() {
  return (
    <div role="status" className="flex min-h-[200px] items-center justify-center">
      <span className="sr-only">Loading…</span>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <ThemeProvider>
            <ToastProvider>
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
            </ToastProvider>
          </ThemeProvider>
        </AuthProvider>
      </BrowserRouter>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}

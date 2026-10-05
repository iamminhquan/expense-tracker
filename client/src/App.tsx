import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { queryClient } from './lib/queryClient'

// Phase 0 placeholder route tree. Real pages (Dashboard, Transactions,
// Categories, Settings, Auth) land in Phase 2 under src/pages/ -- see the
// migration plan for the page-by-page build order.
function Placeholder() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-app text-ink">
      <div className="text-center">
        <p className="wordmark text-3xl">$pend</p>
        <p className="mt-2 text-ink-muted">client/ scaffold is up. Pages land in Phase 2.</p>
      </div>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Placeholder />} />
        </Routes>
      </BrowserRouter>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}

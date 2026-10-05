import { QueryClient } from '@tanstack/react-query'

// Shared TanStack Query client. Phase 1/2 resource hooks (useTransactions,
// useCategories, etc.) all read through this instance so mutations can
// invalidate the right queries -- the replacement for htmx's auto-refresh +
// OOB-swap pattern described in the migration plan.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

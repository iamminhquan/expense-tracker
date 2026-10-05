import { QueryClient } from '@tanstack/react-query'

// Shared TanStack Query client. Every resource hook (useTransactions,
// useCategories, etc.) reads through this one instance so a mutation can
// invalidate whatever else it affects -- see
// .claude/context/frontend.md's Data Layer section.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

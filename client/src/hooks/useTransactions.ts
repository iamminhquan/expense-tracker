import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as transactionsApi from '../lib/api/transactions'
import type { TransactionFilters, TransactionWrite } from '../lib/api/types'

export function transactionsKey(filters: TransactionFilters) {
  return ['transactions', filters] as const
}

export function useTransactions(filters: TransactionFilters) {
  return useQuery({
    queryKey: transactionsKey(filters),
    queryFn: () => transactionsApi.listTransactions(filters),
  })
}

// A transaction write also changes dashboard totals and category counts.
function invalidateEverythingATransactionTouches(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ['transactions'] })
  void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
  void queryClient.invalidateQueries({ queryKey: ['categories'] })
}

export function useCreateTransaction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: transactionsApi.createTransaction,
    onSuccess: () => invalidateEverythingATransactionTouches(queryClient),
  })
}

export function useUpdateTransaction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: TransactionWrite }) => transactionsApi.updateTransaction(id, input),
    onSuccess: () => invalidateEverythingATransactionTouches(queryClient),
  })
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: transactionsApi.deleteTransaction,
    onSuccess: () => invalidateEverythingATransactionTouches(queryClient),
  })
}

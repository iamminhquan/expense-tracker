import { api } from './client'
import type { Transaction, TransactionFilters, TransactionsResponse, TransactionWrite } from './types'

function queryString(filters: TransactionFilters): string {
  const params = new URLSearchParams()
  if (filters.month) params.set('month', filters.month)
  if (filters.page && filters.page > 1) params.set('page', String(filters.page))
  if (filters.q) params.set('q', filters.q)
  if (filters.type) params.set('type', filters.type)
  if (filters.category) params.set('category', String(filters.category))
  if (filters.min) params.set('min', String(filters.min))
  if (filters.max) params.set('max', String(filters.max))
  if (filters.sort) params.set('sort', filters.sort)
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export function listTransactions(filters: TransactionFilters = {}) {
  return api.get<TransactionsResponse>(`/api/v1/transactions${queryString(filters)}`)
}

export function createTransaction(input: TransactionWrite) {
  return api.post<Transaction>('/api/v1/transactions', input)
}

export function updateTransaction(id: number, input: TransactionWrite) {
  return api.patch<Transaction>(`/api/v1/transactions/${id}`, input)
}

export function deleteTransaction(id: number) {
  return api.delete<void>(`/api/v1/transactions/${id}`)
}

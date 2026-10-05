import { api } from './client'
import type { DashboardResponse } from './types'

export function getDashboard(month?: string) {
  const qs = month ? `?month=${encodeURIComponent(month)}` : ''
  return api.get<DashboardResponse>(`/api/dashboard${qs}`)
}

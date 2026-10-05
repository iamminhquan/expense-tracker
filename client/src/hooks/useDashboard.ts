import { useQuery } from '@tanstack/react-query'
import { getDashboard } from '../lib/api/dashboard'

export function dashboardKey(month?: string) {
  return ['dashboard', month ?? 'current'] as const
}

export function useDashboard(month?: string) {
  return useQuery({ queryKey: dashboardKey(month), queryFn: () => getDashboard(month) })
}

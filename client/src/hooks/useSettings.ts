import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as settingsApi from '../lib/api/settings'
import type { Theme } from '../lib/api/types'

export const settingsKey = ['settings'] as const

export function useSettings() {
  return useQuery({ queryKey: settingsKey, queryFn: settingsApi.getSettings })
}

export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.updateProfile,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsKey }),
  })
}

export function useUpdateEmail() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.updateEmail,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsKey }),
  })
}

export function useResendVerification() {
  return useMutation({ mutationFn: settingsApi.resendVerification })
}

export function useUpdatePassword() {
  return useMutation({ mutationFn: settingsApi.updatePassword })
}

export function useDeleteAccount() {
  return useMutation({ mutationFn: settingsApi.deleteAccount })
}

export function useRevokeSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.revokeSession,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsKey }),
  })
}

export function useRevokeOtherSessions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.revokeOtherSessions,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: settingsKey }),
  })
}

export function useUpdateTheme() {
  return useMutation({
    mutationFn: (theme: Theme) => settingsApi.updateTheme(theme),
  })
}

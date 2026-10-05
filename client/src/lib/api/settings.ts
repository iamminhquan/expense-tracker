import { api } from './client'
import type { SettingsResponse, Theme } from './types'

export function getSettings() {
  return api.get<SettingsResponse>('/api/v1/settings')
}

export function updateProfile(input: { name: string; username: string }) {
  return api.patch<void>('/api/v1/settings/profile', input)
}

export function updateEmail(input: { email: string; currentPassword: string }) {
  return api.patch<void>('/api/v1/settings/email', input)
}

export function resendVerification() {
  return api.post<void>('/api/v1/settings/resend-verification')
}

export function updatePassword(input: { currentPassword: string; newPassword: string; newPasswordConfirm: string }) {
  return api.patch<void>('/api/v1/settings/password', input)
}

export function deleteAccount(currentPassword: string) {
  return api.post<void>('/api/v1/settings/delete-account', { currentPassword })
}

export function revokeSession(id: string) {
  return api.delete<void>(`/api/v1/settings/sessions/${encodeURIComponent(id)}`)
}

export function revokeOtherSessions() {
  return api.post<void>('/api/v1/settings/sessions/revoke-others')
}

export function updateTheme(theme: Theme) {
  return api.put<void>('/api/v1/settings/theme', { theme })
}

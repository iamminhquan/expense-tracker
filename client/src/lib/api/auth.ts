import { api } from './client'
import type { AuthResponse, User } from './types'

export interface RegisterInput {
  name: string
  email: string
  username: string
  password: string
  passwordConfirm: string
}

export interface LoginInput {
  email: string
  password: string
}

export function register(input: RegisterInput) {
  return api.post<AuthResponse>('/api/register', input, { skipAuth: true })
}

export function login(input: LoginInput) {
  return api.post<AuthResponse>('/api/login', input, { skipAuth: true })
}

export function logout() {
  return api.post<void>('/api/logout', undefined, { skipAuth: true })
}

export function me() {
  return api.get<User>('/api/me')
}

export function forgotPassword(email: string) {
  return api.post<void>('/api/forgot-password', { email }, { skipAuth: true })
}

export function checkResetToken(token: string) {
  return api.get<void>(`/api/reset-password?token=${encodeURIComponent(token)}`, { skipAuth: true })
}

export function resetPassword(token: string, password: string, passwordConfirm: string) {
  return api.post<AuthResponse>('/api/reset-password', { token, password, passwordConfirm }, { skipAuth: true })
}

export function verifyEmail(token: string) {
  return api.post<{ verified: boolean; conflict: boolean }>('/api/verify-email', { token }, { skipAuth: true })
}

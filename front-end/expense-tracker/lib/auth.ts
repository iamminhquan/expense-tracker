export type AuthUser = {
  id: number;
  name: string;
  email: string;
};

export type AuthResponse = {
  token?: string;
  user?: AuthUser;
  error?: string;
};

export type ProfileResponse = {
  user?: AuthUser;
  error?: string;
};

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

export const AUTH_TOKEN_KEY = "auth_token";

export function getStoredToken() {
  return localStorage.getItem(AUTH_TOKEN_KEY);
}

export function saveToken(token: string) {
  localStorage.setItem(AUTH_TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(AUTH_TOKEN_KEY);
}

export async function fetchProfile(token: string) {
  const response = await fetch(`${API_BASE_URL}/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = (await response.json()) as ProfileResponse;

  return { data, response };
}

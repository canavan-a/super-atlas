import * as SecureStore from 'expo-secure-store';

const SESSION_TOKEN_KEY = 'sa_session_token';

// Set to your server's address; for a physical device this must be a LAN IP
// or public hostname, not localhost.
export const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';
export const WS_BASE_URL = API_BASE_URL.replace(/^http/, 'ws');

// The session token from sign-in. It doesn't expire while the app is in use
// (the server slides its expiry forward), so there's no refresh step: a 401
// only ever means the session was ended for real.
export function getToken(): Promise<string | null> {
  return SecureStore.getItemAsync(SESSION_TOKEN_KEY);
}

export function setToken(token: string): Promise<void> {
  return SecureStore.setItemAsync(SESSION_TOKEN_KEY, token);
}

export function clearToken(): Promise<void> {
  return SecureStore.deleteItemAsync(SESSION_TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  body: any;
  constructor(status: number, message: string, body?: any) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

let onSignedOut: (() => void) | null = null;

// AuthContext registers this so any request that finds the session gone
// drops the app back to sign-in.
export function setSignedOutHandler(handler: (() => void) | null) {
  onSignedOut = handler;
}

export async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const token = await getToken();
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  });
  if (res.status === 401 && token) {
    await clearToken();
    onSignedOut?.();
  }
  return res;
}

export async function apiJson<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await apiFetch(path, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error ?? 'request failed', body);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

import { apiUrl } from '../distribution';
import { tokenStore } from './tokenStore';

export const API_BASE_URL = apiUrl;
export const WS_BASE_URL = API_BASE_URL.replace(/^http/, 'ws');

// The session token from sign-in. It doesn't expire while the app is in use
// (the server slides its expiry forward), so there's no refresh step: a 401
// only ever means the session was ended for real.
export const getToken = tokenStore.get;
export const setToken = tokenStore.set;
export const clearToken = tokenStore.clear;

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

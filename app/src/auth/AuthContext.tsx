import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { apiJson, clearTokens, getTokens, setTokens } from '../api/client';

type User = { id: string; email: string; displayName: string };

type AuthState = {
  status: 'loading' | 'signedOut' | 'signedIn';
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, displayName: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

type AuthResponse = { user: User; accessToken: string; refreshToken: string };

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthState['status']>('loading');
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    (async () => {
      const { accessToken } = await getTokens();
      // We don't have a /me endpoint yet, so presence of a token is enough
      // to attempt the signed-in state; a 401 on first real request will
      // fall back to signedOut via the api client's refresh failure path.
      setStatus(accessToken ? 'signedIn' : 'signedOut');
    })();
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiJson<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    await setTokens(data.accessToken, data.refreshToken);
    setUser(data.user);
    setStatus('signedIn');
  }, []);

  const signup = useCallback(async (email: string, password: string, displayName: string) => {
    const data = await apiJson<AuthResponse>('/auth/signup', {
      method: 'POST',
      body: JSON.stringify({ email, password, displayName }),
    });
    await setTokens(data.accessToken, data.refreshToken);
    setUser(data.user);
    setStatus('signedIn');
  }, []);

  const logout = useCallback(async () => {
    const { refreshToken } = await getTokens();
    if (refreshToken) {
      await apiJson('/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refreshToken }),
      }).catch(() => {});
    }
    await clearTokens();
    setUser(null);
    setStatus('signedOut');
  }, []);

  const value = useMemo(
    () => ({ status, user, login, signup, logout }),
    [status, user, login, signup, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

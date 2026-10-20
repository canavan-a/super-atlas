import AsyncStorage from '@react-native-async-storage/async-storage';
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { ApiError, apiJson, clearToken, getToken, setSignedOutHandler, setToken } from '../api/client';
import { unregisterBackgroundLocationTask } from '../location/backgroundTask';

export type User = { id: string; email: string; displayName: string };

type VerifyResult = 'signedIn' | 'needsName';

type AuthState = {
  status: 'loading' | 'signedOut' | 'signedIn';
  user: User | null;
  ghostMode: boolean;
  requestCode: (email: string) => Promise<void>;
  verify: (email: string, code: string, displayName?: string) => Promise<VerifyResult>;
  setGhostMode: (on: boolean) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

// The signed-in user is cached so a launch without network still opens the
// map; /me refreshes it whenever the server is reachable.
const USER_CACHE_KEY = 'superatlas.user.v1';

type MeResponse = { user: User; ghostMode: boolean };

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthState['status']>('loading');
  const [user, setUser] = useState<User | null>(null);
  const [ghostMode, setGhostModeState] = useState(false);

  const signOutLocally = useCallback(async () => {
    await clearToken();
    await AsyncStorage.removeItem(USER_CACHE_KEY).catch(() => {});
    await unregisterBackgroundLocationTask().catch(() => {});
    setUser(null);
    setGhostModeState(false);
    setStatus('signedOut');
  }, []);

  const refreshMe = useCallback(async () => {
    try {
      const me = await apiJson<MeResponse>('/me');
      setUser(me.user);
      setGhostModeState(me.ghostMode);
      await AsyncStorage.setItem(USER_CACHE_KEY, JSON.stringify(me.user)).catch(() => {});
    } catch {
      // Offline or server down: stay signed in on the cached user. A real
      // 401 is handled by the signed-out handler below.
    }
  }, []);

  useEffect(() => {
    setSignedOutHandler(() => {
      signOutLocally();
    });
    (async () => {
      const token = await getToken();
      if (!token) {
        setStatus('signedOut');
        return;
      }
      const cached = await AsyncStorage.getItem(USER_CACHE_KEY).catch(() => null);
      if (cached) setUser(JSON.parse(cached));
      setStatus('signedIn');
      refreshMe();
    })();
    return () => setSignedOutHandler(null);
  }, [refreshMe, signOutLocally]);

  const requestCode = useCallback(async (email: string) => {
    await apiJson('/auth/code', { method: 'POST', body: JSON.stringify({ email }) });
  }, []);

  const verify = useCallback(
    async (email: string, code: string, displayName?: string): Promise<VerifyResult> => {
      try {
        const data = await apiJson<{ user: User; token: string }>('/auth/verify', {
          method: 'POST',
          body: JSON.stringify({ email, code, displayName }),
        });
        await setToken(data.token);
        await AsyncStorage.setItem(USER_CACHE_KEY, JSON.stringify(data.user)).catch(() => {});
        setUser(data.user);
        setStatus('signedIn');
        refreshMe();
        return 'signedIn';
      } catch (e) {
        if (e instanceof ApiError && e.body?.needsName) return 'needsName';
        throw e;
      }
    },
    [refreshMe]
  );

  const setGhostMode = useCallback(async (on: boolean) => {
    setGhostModeState(on);
    try {
      await apiJson('/locations/pause', { method: 'PUT', body: JSON.stringify({ paused: on }) });
    } catch (e) {
      setGhostModeState(!on);
      throw e;
    }
  }, []);

  const logout = useCallback(async () => {
    await apiJson('/auth/logout', { method: 'POST' }).catch(() => {});
    await signOutLocally();
  }, [signOutLocally]);

  const value = useMemo(
    () => ({ status, user, ghostMode, requestCode, verify, setGhostMode, logout }),
    [status, user, ghostMode, requestCode, verify, setGhostMode, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}

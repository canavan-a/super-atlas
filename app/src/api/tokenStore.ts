import * as SecureStore from 'expo-secure-store';

const SESSION_TOKEN_KEY = 'sa_session_token';

// The session token lives in the Android Keystore-backed secure store.
// (tokenStore.web.ts is the browser's version.)
export const tokenStore = {
  get: (): Promise<string | null> => SecureStore.getItemAsync(SESSION_TOKEN_KEY),
  set: (token: string): Promise<void> => SecureStore.setItemAsync(SESSION_TOKEN_KEY, token),
  clear: (): Promise<void> => SecureStore.deleteItemAsync(SESSION_TOKEN_KEY),
};

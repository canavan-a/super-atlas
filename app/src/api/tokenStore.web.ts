const SESSION_TOKEN_KEY = 'sa_session_token';

// The browser keeps the session token in localStorage: the web build is view
// only, and signing out (or clearing site data) removes it.
export const tokenStore = {
  get: async (): Promise<string | null> => {
    try {
      return localStorage.getItem(SESSION_TOKEN_KEY);
    } catch {
      return null;
    }
  },
  set: async (token: string): Promise<void> => {
    localStorage.setItem(SESSION_TOKEN_KEY, token);
  },
  clear: async (): Promise<void> => {
    try {
      localStorage.removeItem(SESSION_TOKEN_KEY);
    } catch {
      // nothing stored
    }
  },
};

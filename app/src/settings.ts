import AsyncStorage from '@react-native-async-storage/async-storage';

import type { ThemeName } from './theme';

const STORAGE_KEY = 'superatlas.settings.v1';

// Device-local preferences, following super-badger's settings store. There's
// deliberately very little here: the only user-facing choice is the theme.
export interface Settings {
  themeName: ThemeName;
  // Crash-loop protection for the launch splash: a launch counts as a strike
  // before the animation starts and is cleared when it finishes, so a launch
  // that dies mid-animation leaves one behind. After too many in a row the
  // splash turns itself off instead of crashing again.
  splashStrikes: number;
  splashAutoDisabled: boolean;
}

const DEFAULT_SETTINGS: Settings = {
  themeName: 'dark',
  splashStrikes: 0,
  splashAutoDisabled: false,
};

type Listener = (settings: Settings) => void;

let current: Settings = DEFAULT_SETTINGS;
let loaded = false;
const listeners = new Set<Listener>();

async function load(): Promise<Settings> {
  if (loaded) {
    return current;
  }
  try {
    const raw = await AsyncStorage.getItem(STORAGE_KEY);
    if (raw) {
      current = { ...DEFAULT_SETTINGS, ...JSON.parse(raw) };
    }
  } catch {
    // fall back to defaults — a corrupt/missing value shouldn't crash startup
  }
  loaded = true;
  listeners.forEach((l) => l(current));
  return current;
}

async function save(next: Settings): Promise<void> {
  current = next;
  listeners.forEach((l) => l(current));
  try {
    await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    // best-effort persistence; the in-memory value above is still updated
  }
}

export const settingsStore = {
  get: (): Settings => current,
  isLoaded: (): boolean => loaded,
  load,
  save,
  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};

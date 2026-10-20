// Theming: the same five palettes as super-badger and super-chat, switchable
// from Settings. Layout (padding, flex, radii, gaps) stays in each screen's
// StyleSheet — only color tokens come from here, so switching themes never
// has to touch geometry.
import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';

import { settingsStore } from './settings';

export type ThemeName = 'light' | 'dark' | 'slate' | 'sepia' | 'ember';

export interface Theme {
  name: ThemeName;
  // Whether the palette is dark, for the status bar and native controls.
  isDark: boolean;
  bg: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primaryText: string;
  danger: string;
  dangerBg: string;
  success: string;
  // Map markers: friends, and friends whose position is over an hour old.
  friendMarker: string;
  friendMarkerText: string;
  staleMarker: string;
  markerStroke: string;
  // OpenFreeMap basemap that sits best under this palette.
  mapStyleUrl: string;
}

const MAP_LIGHT = 'https://tiles.openfreemap.org/styles/liberty';
const MAP_DARK = 'https://tiles.openfreemap.org/styles/dark';
const MAP_FIORD = 'https://tiles.openfreemap.org/styles/fiord';
const MAP_POSITRON = 'https://tiles.openfreemap.org/styles/positron';

const light: Theme = {
  name: 'light',
  isDark: false,
  bg: '#ffffff',
  surface: '#f6f8fa',
  surfaceAlt: '#f0f0f0',
  border: '#e5e5e5',
  text: '#1b1f24',
  textMuted: '#57606a',
  primary: '#0969da',
  primaryText: '#ffffff',
  danger: '#cf222e',
  dangerBg: '#ffebe9',
  success: '#1a7f37',
  friendMarker: '#0969da',
  friendMarkerText: '#ffffff',
  staleMarker: '#8c959f',
  markerStroke: '#ffffff',
  mapStyleUrl: MAP_LIGHT,
};

const dark: Theme = {
  name: 'dark',
  isDark: true,
  bg: '#0d1117',
  surface: '#161b22',
  surfaceAlt: '#21262d',
  border: '#30363d',
  text: '#e6edf3',
  textMuted: '#8b949e',
  primary: '#2f81f7',
  primaryText: '#ffffff',
  danger: '#f85149',
  dangerBg: '#3d1418',
  success: '#3fb950',
  friendMarker: '#2f81f7',
  friendMarkerText: '#ffffff',
  staleMarker: '#6e7681',
  markerStroke: '#e6edf3',
  mapStyleUrl: MAP_DARK,
};

const slate: Theme = {
  name: 'slate',
  isDark: true,
  bg: '#1e2530',
  surface: '#262e3b',
  surfaceAlt: '#2f3947',
  border: '#3c4757',
  text: '#dde3ea',
  textMuted: '#93a0b3',
  primary: '#5aa9e6',
  primaryText: '#0d1117',
  danger: '#ef6a6a',
  dangerBg: '#3a2222',
  success: '#6fcf97',
  friendMarker: '#5aa9e6',
  friendMarkerText: '#0d1117',
  staleMarker: '#76839a',
  markerStroke: '#dde3ea',
  mapStyleUrl: MAP_FIORD,
};

const sepia: Theme = {
  name: 'sepia',
  isDark: false,
  bg: '#f4ecd8',
  surface: '#ece0c6',
  surfaceAlt: '#e5d7b8',
  border: '#d8c6a0',
  text: '#3b2f1e',
  textMuted: '#7a6a4d',
  primary: '#a0651b',
  primaryText: '#fdf6e7',
  danger: '#a13c2c',
  dangerBg: '#f0dcd2',
  success: '#4e7c3a',
  friendMarker: '#a0651b',
  friendMarkerText: '#fdf6e7',
  staleMarker: '#a8987a',
  markerStroke: '#fdf6e7',
  mapStyleUrl: MAP_POSITRON,
};

// Same dark base as `dark`, swapping its blue accent for a warm red/orange
// one; danger stays a true red so "remove"/errors don't blend into the accent.
const ember: Theme = {
  name: 'ember',
  isDark: true,
  bg: '#050403',
  surface: '#0c0908',
  surfaceAlt: '#150f0d',
  border: '#241815',
  text: '#e8d9d0',
  textMuted: '#8a6c5f',
  primary: '#8a3620',
  primaryText: '#f1e2d9',
  danger: '#c93a3a',
  dangerBg: '#1c0d0c',
  success: '#4f9e70',
  friendMarker: '#b04a2c',
  friendMarkerText: '#f1e2d9',
  staleMarker: '#5e4a41',
  markerStroke: '#e8d9d0',
  mapStyleUrl: MAP_DARK,
};

export const THEMES: Record<ThemeName, Theme> = { light, dark, slate, sepia, ember };

export const THEME_OPTIONS: { name: ThemeName; label: string }[] = [
  { name: 'light', label: 'Light' },
  { name: 'dark', label: 'Dark' },
  { name: 'slate', label: 'Slate' },
  { name: 'sepia', label: 'Sepia' },
  { name: 'ember', label: 'Ember' },
];

const ThemeContext = createContext<{ theme: Theme; setThemeName: (n: ThemeName) => void }>({
  theme: dark,
  setThemeName: () => {},
});

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [themeName, setThemeNameState] = useState<ThemeName>(settingsStore.get().themeName);

  useEffect(() => {
    settingsStore.load().then((s) => setThemeNameState(s.themeName));
    return settingsStore.subscribe((s) => setThemeNameState(s.themeName));
  }, []);

  const value = useMemo(
    () => ({
      theme: THEMES[themeName] ?? dark,
      setThemeName: (n: ThemeName) => settingsStore.save({ ...settingsStore.get(), themeName: n }),
    }),
    [themeName]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext).theme;
}

export function useThemeSetting() {
  const { theme, setThemeName } = useContext(ThemeContext);
  return { theme, themeName: theme.name, setThemeName };
}

// Builds a StyleSheet from the current theme, rebuilt only when it changes.
export function useThemedStyles<T>(make: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => make(theme), [theme]); // eslint-disable-line react-hooks/exhaustive-deps
}

import { useEffect, useRef } from 'react';
import { AppState, Platform } from 'react-native';

import AppIcon from '../modules/app-icon';
import type { ThemeName } from './theme';
import { useTheme } from './theme';

// The launcher icon follows the theme, as in super-badger and super-chat.
// Android has no API to swap an app's icon directly; instead the manifest
// declares one launcher entry per theme (an <activity-alias>, each with its
// own icon, added by plugins/withAppIcon.js) and exactly one is enabled at a
// time — see modules/app-icon. Elsewhere this is a no-op.

/** The theme every install starts on, i.e. the alias enabled in the manifest. */
export const DEFAULT_ICON: ThemeName = 'dark';

async function setAppIcon(name: ThemeName): Promise<boolean> {
  if (Platform.OS !== 'android' || !AppIcon) return false;
  try {
    return await AppIcon.setIcon(name);
  } catch {
    return false; // a launcher hiccup must never take the app down
  }
}

async function getAppIcon(): Promise<ThemeName | null> {
  if (Platform.OS !== 'android' || !AppIcon) return null;
  try {
    return (await AppIcon.getIcon()) as ThemeName;
  } catch {
    return null;
  }
}

/**
 * Keeps the launcher icon in step with the theme, but only ever changes it as
 * the app goes to the background. Switching while the app is open can make
 * some launchers flicker, drop a pinned shortcut, or restart the app; done
 * while the user has already left, none of that is visible.
 */
export function useThemedAppIcon(): void {
  const theme = useTheme();
  const want = useRef<ThemeName>(theme.name);
  want.current = theme.name;
  // what the launcher is showing now (null until we've asked)
  const applied = useRef<ThemeName | null>(null);

  useEffect(() => {
    if (Platform.OS !== 'android') return;
    let alive = true;
    getAppIcon().then((cur) => {
      if (alive) applied.current = cur;
    });
    const sub = AppState.addEventListener('change', (state) => {
      if (state !== 'background') return;
      const target = want.current;
      if (applied.current === target) return;
      setAppIcon(target).then((ok) => {
        if (ok) applied.current = target;
      });
    });
    return () => {
      alive = false;
      sub.remove();
    };
  }, []);
}

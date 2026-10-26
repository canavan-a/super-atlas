import * as Location from 'expo-location';
import { useCallback, useEffect, useState } from 'react';
import { AppState, Linking } from 'react-native';
import { registerBackgroundLocationTask } from './backgroundTask';

export type PermissionsState = {
  loading: boolean;
  // Required before the app proceeds past onboarding.
  foregroundLocationGranted: boolean;
  // Optional: without it the app works, but friends only see you while it's
  // open, and BackgroundLocationBanner nags about it.
  backgroundLocationGranted: boolean;
  // The OS won't show its foreground prompt again, so the only way forward
  // is Settings.
  mustOpenSettings: boolean;
  // Asks for foreground location, then background (onboarding).
  requestAll: () => Promise<void>;
  // Asks for background location alone, or opens Settings once the OS has
  // stopped showing its prompt (the banner).
  requestBackground: () => Promise<void>;
};

export function usePermissionsGate(): PermissionsState {
  const [loading, setLoading] = useState(true);
  const [foregroundLocationGranted, setForegroundLocationGranted] = useState(false);
  const [backgroundLocationGranted, setBackgroundLocationGranted] = useState(false);
  const [mustOpenSettings, setMustOpenSettings] = useState(false);
  const [backgroundMustOpenSettings, setBackgroundMustOpenSettings] = useState(false);

  const refresh = useCallback(async () => {
    const [fg, bg] = await Promise.all([
      Location.getForegroundPermissionsAsync(),
      Location.getBackgroundPermissionsAsync(),
    ]);
    const fgGranted = fg.status === Location.PermissionStatus.GRANTED;
    const bgGranted = bg.status === Location.PermissionStatus.GRANTED;
    setForegroundLocationGranted(fgGranted);
    setBackgroundLocationGranted(bgGranted);
    setMustOpenSettings(!fgGranted && !fg.canAskAgain);
    setBackgroundMustOpenSettings(!bgGranted && !bg.canAskAgain);
    setLoading(false);
  }, []);

  // Re-check whenever the app comes back, e.g. from the Settings app.
  useEffect(() => {
    refresh();
    const sub = AppState.addEventListener('change', (state) => {
      if (state === 'active') refresh();
    });
    return () => sub.remove();
  }, [refresh]);

  const askBackground = async () => {
    const bg = await Location.requestBackgroundPermissionsAsync();
    if (bg.status === Location.PermissionStatus.GRANTED) {
      await registerBackgroundLocationTask();
    }
  };

  const requestAll = useCallback(async () => {
    if (mustOpenSettings) {
      await Linking.openSettings();
      return;
    }
    // Foreground location must be granted before background can even be
    // requested on both iOS and Android.
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status === Location.PermissionStatus.GRANTED) await askBackground();
    await refresh();
  }, [mustOpenSettings, refresh]);

  const requestBackground = useCallback(async () => {
    if (backgroundMustOpenSettings) {
      await Linking.openSettings();
      return;
    }
    await askBackground();
    await refresh();
  }, [backgroundMustOpenSettings, refresh]);

  return {
    loading,
    foregroundLocationGranted,
    backgroundLocationGranted,
    mustOpenSettings,
    requestAll,
    requestBackground,
  };
}

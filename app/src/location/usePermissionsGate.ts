import * as Location from 'expo-location';
import { useCallback, useEffect, useState } from 'react';
import { AppState, Linking } from 'react-native';
import { registerBackgroundLocationTask } from './backgroundTask';

export type PermissionsState = {
  loading: boolean;
  // Both are required before the app proceeds past onboarding.
  foregroundLocationGranted: boolean;
  backgroundLocationGranted: boolean;
  allGranted: boolean;
  // The OS won't show its prompt again, so the only way forward is Settings.
  mustOpenSettings: boolean;
  requestAll: () => Promise<void>;
};

export function usePermissionsGate(): PermissionsState {
  const [loading, setLoading] = useState(true);
  const [foregroundLocationGranted, setForegroundLocationGranted] = useState(false);
  const [backgroundLocationGranted, setBackgroundLocationGranted] = useState(false);
  const [mustOpenSettings, setMustOpenSettings] = useState(false);

  const refresh = useCallback(async () => {
    const [fg, bg] = await Promise.all([
      Location.getForegroundPermissionsAsync(),
      Location.getBackgroundPermissionsAsync(),
    ]);
    const fgGranted = fg.status === Location.PermissionStatus.GRANTED;
    const bgGranted = bg.status === Location.PermissionStatus.GRANTED;
    setForegroundLocationGranted(fgGranted);
    setBackgroundLocationGranted(bgGranted);
    setMustOpenSettings((!fgGranted && !fg.canAskAgain) || (fgGranted && !bgGranted && !bg.canAskAgain));
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

  const requestAll = useCallback(async () => {
    if (mustOpenSettings) {
      await Linking.openSettings();
      return;
    }
    // Foreground location must be granted before background can even be
    // requested on both iOS and Android.
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status === Location.PermissionStatus.GRANTED) {
      const bg = await Location.requestBackgroundPermissionsAsync();
      if (bg.status === Location.PermissionStatus.GRANTED) {
        await registerBackgroundLocationTask();
      }
    }
    await refresh();
  }, [mustOpenSettings, refresh]);

  return {
    loading,
    foregroundLocationGranted,
    backgroundLocationGranted,
    allGranted: foregroundLocationGranted && backgroundLocationGranted,
    mustOpenSettings,
    requestAll,
  };
}

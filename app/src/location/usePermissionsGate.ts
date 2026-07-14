import * as Location from 'expo-location';
import { useCallback, useEffect, useState } from 'react';
import { registerBackgroundLocationTask } from './backgroundTask';

export type PermissionsState = {
  loading: boolean;
  // Both are required before the app proceeds past onboarding.
  foregroundLocationGranted: boolean;
  backgroundLocationGranted: boolean;
  allGranted: boolean;
  requestAll: () => Promise<void>;
};

export function usePermissionsGate(): PermissionsState {
  const [loading, setLoading] = useState(true);
  const [foregroundLocationGranted, setForegroundLocationGranted] = useState(false);
  const [backgroundLocationGranted, setBackgroundLocationGranted] = useState(false);

  const refresh = useCallback(async () => {
    const [fg, bg] = await Promise.all([
      Location.getForegroundPermissionsAsync(),
      Location.getBackgroundPermissionsAsync(),
    ]);
    setForegroundLocationGranted(fg.status === Location.PermissionStatus.GRANTED);
    setBackgroundLocationGranted(bg.status === Location.PermissionStatus.GRANTED);
    setLoading(false);
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const requestAll = useCallback(async () => {
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
  }, [refresh]);

  return {
    loading,
    foregroundLocationGranted,
    backgroundLocationGranted,
    allGranted: foregroundLocationGranted && backgroundLocationGranted,
    requestAll,
  };
}

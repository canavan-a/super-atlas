import * as Location from 'expo-location';
import * as Notifications from 'expo-notifications';
import { useCallback, useEffect, useState } from 'react';
import { registerBackgroundLocationTask } from './backgroundTask';

export type PermissionsState = {
  loading: boolean;
  // All three are required before the app proceeds past onboarding.
  foregroundLocationGranted: boolean;
  backgroundLocationGranted: boolean;
  notificationsGranted: boolean;
  allGranted: boolean;
  requestAll: () => Promise<void>;
};

export function usePermissionsGate(): PermissionsState {
  const [loading, setLoading] = useState(true);
  const [foregroundLocationGranted, setForegroundLocationGranted] = useState(false);
  const [backgroundLocationGranted, setBackgroundLocationGranted] = useState(false);
  const [notificationsGranted, setNotificationsGranted] = useState(false);

  const refresh = useCallback(async () => {
    const [fg, bg, notif] = await Promise.all([
      Location.getForegroundPermissionsAsync(),
      Location.getBackgroundPermissionsAsync(),
      Notifications.getPermissionsAsync(),
    ]);
    setForegroundLocationGranted(fg.status === Location.PermissionStatus.GRANTED);
    setBackgroundLocationGranted(bg.status === Location.PermissionStatus.GRANTED);
    setNotificationsGranted(notif.status === Notifications.PermissionStatus.GRANTED);
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
    await Notifications.requestPermissionsAsync();
    await refresh();
  }, [refresh]);

  return {
    loading,
    foregroundLocationGranted,
    backgroundLocationGranted,
    notificationsGranted,
    allGranted: foregroundLocationGranted && backgroundLocationGranted && notificationsGranted,
    requestAll,
  };
}

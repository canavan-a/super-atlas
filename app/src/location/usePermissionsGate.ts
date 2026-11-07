import * as Location from 'expo-location';
import { useCallback, useEffect, useState } from 'react';
import { AppState, Linking, Platform } from 'react-native';
import { registerBackgroundLocationTask } from './backgroundTask';

export type PermissionsState = {
  loading: boolean;
  // Whether to show PermissionsGateScreen. Android: until location is
  // granted. iOS: until the location prompt has been answered at all, since
  // App Review (5.1.1(iv)) won't have the app locked to someone who said
  // no; they get the map without sharing, and BackgroundLocationBanner
  // points them to Settings.
  showGate: boolean;
  foregroundLocationGranted: boolean;
  // Optional: without it the app works, but friends only see you while it's
  // open, and BackgroundLocationBanner nags about it.
  backgroundLocationGranted: boolean;
  // iOS: Precise Location is switched off, so friends only see roughly
  // where you are (and iOS won't monitor the region motion tracking uses).
  preciseLocationOff: boolean;
  // The OS won't show its foreground prompt again, so the only way forward
  // is Settings.
  mustOpenSettings: boolean;
  // The same for the background prompt.
  backgroundMustOpenSettings: boolean;
  // Asks for foreground location, then background (onboarding).
  requestAll: () => Promise<void>;
  // Asks for background location alone, or opens Settings once the OS has
  // stopped showing its prompt (the banner).
  requestBackground: () => Promise<void>;
};

export function usePermissionsGate(): PermissionsState {
  const [loading, setLoading] = useState(true);
  const [showGate, setShowGate] = useState(true);
  const [foregroundLocationGranted, setForegroundLocationGranted] = useState(false);
  const [backgroundLocationGranted, setBackgroundLocationGranted] = useState(false);
  const [preciseLocationOff, setPreciseLocationOff] = useState(false);
  const [mustOpenSettings, setMustOpenSettings] = useState(false);
  const [backgroundMustOpenSettings, setBackgroundMustOpenSettings] = useState(false);

  const refresh = useCallback(async () => {
    const [fg, bg] = await Promise.all([
      Location.getForegroundPermissionsAsync(),
      Location.getBackgroundPermissionsAsync(),
    ]);
    const fgGranted = fg.status === Location.PermissionStatus.GRANTED;
    const bgGranted = bg.status === Location.PermissionStatus.GRANTED;
    setShowGate(Platform.OS === 'ios' ? fg.status === Location.PermissionStatus.UNDETERMINED : !fgGranted);
    setForegroundLocationGranted(fgGranted);
    setBackgroundLocationGranted(bgGranted);
    setPreciseLocationOff(fgGranted && fg.ios?.accuracy === 'reduced');
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
    // requested on both iOS and Android. On iOS this is "While Using" first,
    // then the "Change to Always Allow" prompt, which iOS shows straight
    // away when asked right after (Apple's recommended order).
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status === Location.PermissionStatus.GRANTED) await askBackground();
    await refresh();
  }, [mustOpenSettings, refresh]);

  const requestBackground = useCallback(async () => {
    if (backgroundMustOpenSettings || mustOpenSettings) {
      await Linking.openSettings();
      return;
    }
    if (!foregroundLocationGranted) {
      await requestAll();
      return;
    }
    await askBackground();
    await refresh();
  }, [backgroundMustOpenSettings, mustOpenSettings, foregroundLocationGranted, requestAll, refresh]);

  return {
    loading,
    showGate,
    foregroundLocationGranted,
    backgroundLocationGranted,
    preciseLocationOff,
    mustOpenSettings,
    backgroundMustOpenSettings,
    requestAll,
    requestBackground,
  };
}

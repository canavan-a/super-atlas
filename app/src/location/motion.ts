import AsyncStorage from '@react-native-async-storage/async-storage';
import { NativeModules, PermissionsAndroid, Platform } from 'react-native';
import { getToken } from '../api/client';

// Motion-triggered sharing: near-live updates while you're moving, even with
// the app closed, and nothing running while you're still. Each platform
// implements it natively behind the same "Motion" module, so this file is
// the only JS that knows about it:
//  Android  MotionTracker.kt: a geofence where you stopped, plus Activity
//           Recognition (car, bike) for a quicker start, wake a location
//           foreground service that sends a fix every few seconds until
//           you've been still for 5 minutes.
//  iOS      not built yet (no ios/ project; see TODO.md): region monitoring
//           and significant-change monitoring wake the app, CoreMotion's
//           CMMotionActivityManager confirms the trip and the stop, with
//           the same enable/disable methods.
// The web build never shares a location and doesn't import this.
interface MotionNative {
  enable(token: string): Promise<void>;
  disable(): Promise<void>;
}

const Motion = NativeModules.Motion as MotionNative | undefined;

/** Turns it on as the current session (or re-arms it). Safe to repeat. */
export async function startMotionTracking() {
  const token = await getToken();
  if (token) await Motion?.enable(token);
}

export async function stopMotionTracking() {
  await Motion?.disable();
}

const ACTIVITY_ASKED_KEY = 'superatlas.activityAsked.v1';

/**
 * Asks for the permission that lets a trip start tracking within a minute
 * ("Physical activity" on Android, "Motion & Fitness" on iOS). Optional:
 * without it the geofence still starts tracking, just a couple of minutes
 * later. Returns whether it's granted.
 */
export async function requestActivityPermission(): Promise<boolean> {
  let granted = false;
  if (Platform.OS === 'android') {
    // Granted at install before Android 10.
    granted =
      Platform.Version < 29 ||
      (await PermissionsAndroid.request(PermissionsAndroid.PERMISSIONS.ACTIVITY_RECOGNITION)) ===
        PermissionsAndroid.RESULTS.GRANTED;
  }
  await AsyncStorage.setItem(ACTIVITY_ASKED_KEY, '1').catch(() => {});
  // Re-arm, so the native side starts listening for trips.
  if (granted) await startMotionTracking();
  return granted;
}

/** Asks once, for people who were set up before this existed. */
export async function requestActivityPermissionOnce() {
  if (await AsyncStorage.getItem(ACTIVITY_ASKED_KEY).catch(() => '1')) return;
  await requestActivityPermission();
}

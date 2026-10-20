// Friend notifications ("<name> wants to be friends", "<name> is now your
// friend"), delivered differently per build (src/distribution.ts):
//  sideload  an opt-in native foreground service holding a connection to the
//            server (NotifyService.kt), switched from Settings.
//  play      Firebase Cloud Messaging: this device's token is registered
//            with the server after sign-in (PushModule.kt).
//  web       none.
import { NativeModules, PermissionsAndroid, Platform } from 'react-native';

import { apiJson, getToken } from './api/client';
import { apiUrl, distribution } from './distribution';

interface NotifyNative {
  isEnabled(): Promise<boolean>;
  enable(apiUrl: string, token: string): Promise<void>;
  disable(): Promise<void>;
}

interface PushNative {
  isAvailable(): Promise<boolean>;
  getToken(): Promise<string>;
}

const Notify = NativeModules.Notify as NotifyNative | undefined;
const Push = NativeModules.Push as PushNative | undefined;

/** Whether this build has the Settings switch for background notifications. */
export const hasBackgroundNotificationSwitch = distribution === 'sideload' && Notify != null;

// Android 13+ needs the user's OK before an app may notify at all.
async function allowNotifications(): Promise<boolean> {
  if (Platform.OS !== 'android' || Platform.Version < 33) return true;
  const result = await PermissionsAndroid.request(PermissionsAndroid.PERMISSIONS.POST_NOTIFICATIONS);
  return result === PermissionsAndroid.RESULTS.GRANTED;
}

export async function backgroundNotificationsEnabled(): Promise<boolean> {
  return Notify ? Notify.isEnabled().catch(() => false) : false;
}

/** Sideload only. Returns whether they ended up on. */
export async function setBackgroundNotifications(on: boolean): Promise<boolean> {
  if (!Notify) return false;
  if (!on) {
    await Notify.disable();
    return false;
  }
  if (!(await allowNotifications())) return false;
  const token = await getToken();
  if (!token) return false;
  await Notify.enable(apiUrl, token);
  return true;
}

/**
 * Called once signed in. Play: register this device for pushes. Sideload: if
 * background notifications are on, reconnect them as the current session.
 */
export async function startNotifications(): Promise<void> {
  if (distribution === 'play' && Push) {
    if (!(await Push.isAvailable().catch(() => false))) return;
    if (!(await allowNotifications())) return;
    const token = await Push.getToken();
    await apiJson('/push/token', { method: 'PUT', body: JSON.stringify({ token }) });
  } else if (distribution === 'sideload' && Notify && (await backgroundNotificationsEnabled())) {
    const token = await getToken();
    if (token) await Notify.enable(apiUrl, token);
  }
}

/** Called on sign-out. (The server drops this session's push token itself.) */
export async function stopNotifications(): Promise<void> {
  await Notify?.disable().catch(() => {});
}

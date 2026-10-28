import * as BackgroundTask from 'expo-background-task';
import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import { NativeModules } from 'react-native';
import { apiJson, getToken } from '../api/client';

export const BACKGROUND_LOCATION_TASK = 'superatlas-background-location';

// Motion-triggered sharing (android/.../MotionTracker.kt): a geofence where
// you stopped, and a foreground service sending fixes while you're moving
// once you leave it. Native, because only a native geofence receiver may
// start a location service with the app closed. The OS-scheduled task below
// is the fallback: it covers phones without Google Play services, and re-arms
// the geofence whenever it runs in case Android dropped it.
interface MotionNative {
  enable(token: string): Promise<void>;
  disable(): Promise<void>;
}

const Motion = NativeModules.Motion as MotionNative | undefined;

async function startMotionTracking() {
  const token = await getToken();
  if (token) await Motion?.enable(token);
}

// Runs on the OS's own schedule (typically every 15-30+ min, batched with
// other apps' background work for battery efficiency) rather than on a timer
// we control. We grab one fix and upload it — this is the "conserve battery
// in background" path; the WebSocket stream (useLocationStream) handles foreground.
TaskManager.defineTask(BACKGROUND_LOCATION_TASK, async () => {
  try {
    const { status } = await Location.getBackgroundPermissionsAsync();
    if (status !== Location.PermissionStatus.GRANTED) {
      return BackgroundTask.BackgroundTaskResult.Failed;
    }

    await startMotionTracking().catch(() => {});

    const position = await Location.getCurrentPositionAsync({
      accuracy: Location.Accuracy.Balanced,
    });

    await apiJson('/locations/batch', {
      method: 'POST',
      body: JSON.stringify({
        points: [
          {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            accuracyM: position.coords.accuracy ?? undefined,
            recordedAt: new Date(position.timestamp).toISOString(),
          },
        ],
      }),
    });

    return BackgroundTask.BackgroundTaskResult.Success;
  } catch {
    return BackgroundTask.BackgroundTaskResult.Failed;
  }
});

export async function registerBackgroundLocationTask() {
  await BackgroundTask.registerTaskAsync(BACKGROUND_LOCATION_TASK, {
    minimumInterval: 15, // minutes; OS treats this as a floor, not a guarantee
  });
  await startMotionTracking();
}

export async function unregisterBackgroundLocationTask() {
  await Motion?.disable();
  const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK);
  if (isRegistered) {
    await BackgroundTask.unregisterTaskAsync(BACKGROUND_LOCATION_TASK);
  }
}

// Registers the task and turns on motion tracking if background permission
// is granted. Called on every signed-in launch, since a reinstall, an update
// or signing in again can leave them off even though the permission is
// still there.
export async function ensureBackgroundLocationTask() {
  const { status } = await Location.getBackgroundPermissionsAsync();
  if (status !== Location.PermissionStatus.GRANTED) return;
  if (!(await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK))) {
    await BackgroundTask.registerTaskAsync(BACKGROUND_LOCATION_TASK, { minimumInterval: 15 });
  }
  await startMotionTracking();
}

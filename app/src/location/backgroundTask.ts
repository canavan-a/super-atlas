import * as BackgroundTask from 'expo-background-task';
import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import { apiJson } from '../api/client';

export const BACKGROUND_LOCATION_TASK = 'superatlas-background-location';

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
}

export async function unregisterBackgroundLocationTask() {
  const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK);
  if (isRegistered) {
    await BackgroundTask.unregisterTaskAsync(BACKGROUND_LOCATION_TASK);
  }
}

// Registers the task if background permission is granted and it isn't
// registered yet. Called on every signed-in launch, since a reinstall, an
// update or signing in again can leave it unregistered even though the
// permission is still there.
export async function ensureBackgroundLocationTask() {
  const { status } = await Location.getBackgroundPermissionsAsync();
  if (status !== Location.PermissionStatus.GRANTED) return;
  if (await TaskManager.isTaskRegisteredAsync(BACKGROUND_LOCATION_TASK)) return;
  await registerBackgroundLocationTask();
}

import * as Location from 'expo-location';
import { useEffect, useState } from 'react';
import { FriendLocation, useFriendStream } from './friendStream';

export type { FriendLocation };

export type MyPosition = { latitude: number; longitude: number; accuracyM: number | null };

// Friends' positions in, and this device's position out over the same
// socket while the app is open. The same fixes draw your own dot on the map,
// so you see exactly what your friends see. (useLocationStream.web.ts only
// receives.)
export function useLocationStream(): { locations: FriendLocation[]; me: MyPosition | null } {
  const { locations, socket } = useFriendStream();
  const [me, setMe] = useState<MyPosition | null>(null);

  useEffect(() => {
    let stopped = false;
    let watcher: Location.LocationSubscription | null = null;
    const show = (position: Location.LocationObject) =>
      setMe({
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        accuracyM: position.coords.accuracy,
      });
    (async () => {
      // The OS's cached fix, so the dot shows straight away instead of after
      // the first fresh one.
      const last = await Location.getLastKnownPositionAsync().catch(() => null);
      if (last && !stopped) show(last);
      // GPS every few seconds while the app is open, so friends watching see
      // you move near live (the screen is on anyway, so it costs little
      // extra). Closed, MotionTracker.kt takes over while you're moving.
      watcher = await Location.watchPositionAsync(
        { accuracy: Location.Accuracy.High, distanceInterval: 5, timeInterval: 3000 },
        (position) => {
          show(position);
          const ws = socket.current;
          if (ws?.readyState !== WebSocket.OPEN) return;
          ws.send(
            JSON.stringify({
              latitude: position.coords.latitude,
              longitude: position.coords.longitude,
              accuracyM: position.coords.accuracy ?? undefined,
              recordedAt: new Date(position.timestamp).toISOString(),
            })
          );
        }
      );
      if (stopped) watcher.remove();
    })();
    return () => {
      stopped = true;
      watcher?.remove();
    };
  }, [socket]);

  return { locations, me };
}

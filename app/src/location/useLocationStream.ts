import * as Location from 'expo-location';
import { useEffect } from 'react';
import { FriendLocation, useFriendStream } from './friendStream';

export type { FriendLocation };

// Friends' positions in, and this device's position out over the same
// socket while the app is open. (useLocationStream.web.ts only receives.)
export function useLocationStream(): FriendLocation[] {
  const { locations, socket } = useFriendStream();

  useEffect(() => {
    let stopped = false;
    let watcher: Location.LocationSubscription | null = null;
    (async () => {
      watcher = await Location.watchPositionAsync(
        { accuracy: Location.Accuracy.Balanced, distanceInterval: 15, timeInterval: 10000 },
        (position) => {
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

  return locations;
}

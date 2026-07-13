import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';
import { WS_BASE_URL, getTokens } from '../api/client';

export type FriendLocation = {
  userId: string;
  latitude: number;
  longitude: number;
  accuracyM?: number;
  recordedAt: string;
};

// Keeps a WebSocket open while the screen is mounted (i.e. app foregrounded
// on the map screen): streams this device's position out, and merges
// incoming friend positions into local state for the map to render.
export function useLocationStream() {
  const [friendLocations, setFriendLocations] = useState<Record<string, FriendLocation>>({});
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let cancelled = false;
    let watcher: Location.LocationSubscription | null = null;

    (async () => {
      const { accessToken } = await getTokens();
      if (!accessToken || cancelled) return;

      const ws = new WebSocket(
        `${WS_BASE_URL}/locations/stream?access_token=${encodeURIComponent(accessToken)}`
      );
      wsRef.current = ws;

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'location_update') {
            setFriendLocations((prev) => ({
              ...prev,
              [data.userId]: {
                userId: data.userId,
                latitude: data.latitude,
                longitude: data.longitude,
                accuracyM: data.accuracyM,
                recordedAt: data.recordedAt,
              },
            }));
          }
        } catch {
          // ignore malformed frames
        }
      };

      watcher = await Location.watchPositionAsync(
        { accuracy: Location.Accuracy.Balanced, distanceInterval: 15, timeInterval: 10000 },
        (position) => {
          if (ws.readyState !== WebSocket.OPEN) return;
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
    })();

    return () => {
      cancelled = true;
      watcher?.remove();
      wsRef.current?.close();
    };
  }, []);

  return Object.values(friendLocations);
}

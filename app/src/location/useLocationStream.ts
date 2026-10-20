import * as Location from 'expo-location';
import { useEffect, useState } from 'react';
import { AppState } from 'react-native';
import { WS_BASE_URL, getToken } from '../api/client';

export type FriendLocation = {
  userId: string;
  displayName: string;
  latitude: number;
  longitude: number;
  accuracyM?: number;
  recordedAt: string;
};

const MIN_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;

function toFriendLocation(data: any): FriendLocation {
  return {
    userId: data.userId,
    displayName: data.displayName ?? '',
    latitude: data.latitude,
    longitude: data.longitude,
    accuracyM: data.accuracyM,
    recordedAt: data.recordedAt,
  };
}

// Keeps a WebSocket open while the app is in the foreground: streams this
// device's position out, and merges friends' positions in. The socket
// reconnects on its own (with backoff) after a drop, and right away when the
// app comes back to the foreground; each connect starts with a fresh
// snapshot, so anyone who went into ghost mode meanwhile disappears.
export function useLocationStream() {
  const [friendLocations, setFriendLocations] = useState<Record<string, FriendLocation>>({});

  useEffect(() => {
    let stopped = false;
    let ws: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let retryMs = MIN_RETRY_MS;
    let watcher: Location.LocationSubscription | null = null;

    const clearRetry = () => {
      if (retryTimer) clearTimeout(retryTimer);
      retryTimer = null;
    };

    const scheduleReconnect = () => {
      if (stopped || retryTimer || AppState.currentState !== 'active') return;
      retryTimer = setTimeout(() => {
        retryTimer = null;
        connect();
      }, retryMs);
      retryMs = Math.min(retryMs * 2, MAX_RETRY_MS);
    };

    const connect = async () => {
      if (stopped || ws) return;
      const token = await getToken();
      if (!token || stopped || ws) return;

      const socket = new WebSocket(`${WS_BASE_URL}/locations/stream?token=${encodeURIComponent(token)}`);
      ws = socket;
      socket.onopen = () => {
        retryMs = MIN_RETRY_MS;
      };
      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'snapshot') {
            const next: Record<string, FriendLocation> = {};
            for (const l of data.locations ?? []) next[l.userId] = toFriendLocation(l);
            setFriendLocations(next);
          } else if (data.type === 'location_update') {
            setFriendLocations((prev) => ({ ...prev, [data.userId]: toFriendLocation(data) }));
          }
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        if (ws === socket) ws = null;
        scheduleReconnect();
      };
    };

    const disconnect = () => {
      clearRetry();
      const socket = ws;
      ws = null;
      socket?.close();
    };

    const appStateSub = AppState.addEventListener('change', (state) => {
      if (state === 'active') {
        clearRetry();
        retryMs = MIN_RETRY_MS;
        connect();
      } else {
        // The background task covers updates while the app is closed.
        disconnect();
      }
    });

    connect();
    (async () => {
      watcher = await Location.watchPositionAsync(
        { accuracy: Location.Accuracy.Balanced, distanceInterval: 15, timeInterval: 10000 },
        (position) => {
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
      appStateSub.remove();
      watcher?.remove();
      disconnect();
    };
  }, []);

  return Object.values(friendLocations);
}

import { useEffect, useRef, useState } from 'react';
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

// Keeps the location WebSocket open while the app is in the foreground and
// merges friends' positions in. It reconnects on its own (with backoff) after
// a drop, and right away when the app comes back to the foreground; each
// connect starts with a fresh snapshot, so anyone who went into ghost mode
// meanwhile disappears. Receiving only: useLocationStream adds sending this
// device's position on Android (the web build never sends one).
export function useFriendStream() {
  const [friendLocations, setFriendLocations] = useState<Record<string, FriendLocation>>({});
  const socket = useRef<WebSocket | null>(null);

  useEffect(() => {
    let stopped = false;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let retryMs = MIN_RETRY_MS;

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
      if (stopped || socket.current) return;
      const token = await getToken();
      if (!token || stopped || socket.current) return;

      const ws = new WebSocket(`${WS_BASE_URL}/locations/stream?token=${encodeURIComponent(token)}`);
      socket.current = ws;
      ws.onopen = () => {
        retryMs = MIN_RETRY_MS;
      };
      ws.onmessage = (event) => {
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
      ws.onclose = () => {
        if (socket.current === ws) socket.current = null;
        scheduleReconnect();
      };
    };

    const disconnect = () => {
      clearRetry();
      const ws = socket.current;
      socket.current = null;
      ws?.close();
    };

    const appStateSub = AppState.addEventListener('change', (state) => {
      if (state === 'active') {
        clearRetry();
        retryMs = MIN_RETRY_MS;
        connect();
      } else {
        // On Android the background task covers updates while the app is
        // closed; on the web nothing is shared anyway.
        disconnect();
      }
    });

    connect();
    return () => {
      stopped = true;
      appStateSub.remove();
      disconnect();
    };
  }, []);

  return { locations: Object.values(friendLocations), socket };
}

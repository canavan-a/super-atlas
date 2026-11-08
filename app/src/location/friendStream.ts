import { useEffect, useRef, useState } from 'react';
import { AppState } from 'react-native';
import { WS_BASE_URL, getToken } from '../api/client';

export type FriendLocation = {
  userId: string;
  displayName: string;
  avatarId?: string;
  latitude: number;
  longitude: number;
  accuracyM?: number;
  // How fast they were going (when their phone says), and since when
  // they've been near this spot.
  speedMps?: number;
  recordedAt: string;
  hereSince?: string;
};

const MIN_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;

function toFriendLocation(data: any): FriendLocation {
  return {
    userId: data.userId,
    displayName: data.displayName ?? '',
    avatarId: data.avatarId || undefined,
    latitude: data.latitude,
    longitude: data.longitude,
    accuracyM: data.accuracyM,
    speedMps: data.speedMps,
    recordedAt: data.recordedAt,
    hereSince: data.hereSince,
  };
}

// Keeps the location WebSocket open while the app is in the foreground and
// merges friends' positions in. It reconnects on its own (with backoff) after
// a drop, and right away when the app comes back to the foreground. Anyone
// who goes into ghost mode drops off as they do, and each
// connect starts with a fresh snapshot that leaves out anyone who did so
// meanwhile. Receiving only: useLocationStream adds sending this
// device's position on Android (the web build never sends one). `own` is
// your account's last position as the server has it (your phone's), which
// the web map shows as you; the phone draws its own GPS fix instead.
export function useFriendStream() {
  const [friendLocations, setFriendLocations] = useState<Record<string, FriendLocation>>({});
  const [own, setOwn] = useState<FriendLocation | null>(null);
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
            setOwn(data.me ? toFriendLocation(data.me) : null);
          } else if (data.type === 'self_update') {
            setOwn(toFriendLocation(data));
          } else if (data.type === 'location_update') {
            setFriendLocations((prev) => ({ ...prev, [data.userId]: toFriendLocation(data) }));
          } else if (data.type === 'location_removed') {
            // They went into ghost mode.
            setFriendLocations((prev) => {
              const { [data.userId]: _gone, ...rest } = prev;
              return rest;
            });
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

  return { locations: Object.values(friendLocations), own, socket };
}

import { useMemo } from 'react';
import { FriendLocation, useFriendStream } from './friendStream';
import type { MyPosition } from './useLocationStream';

export type { FriendLocation };

// The web build is view only: it never asks for or sends this browser's
// location. "Me" here is your phone's last position, as the server has it.
export function useLocationStream(): { locations: FriendLocation[]; me: MyPosition | null } {
  const { locations, own } = useFriendStream();
  const me = useMemo(
    () =>
      own
        ? { latitude: own.latitude, longitude: own.longitude, accuracyM: own.accuracyM ?? null, recordedAt: own.recordedAt }
        : null,
    [own]
  );
  return { locations, me };
}

import { FriendLocation, useFriendStream } from './friendStream';
import type { MyPosition } from './useLocationStream';

export type { FriendLocation };

// The web build is view only: it shows friends' positions and never sends
// this browser's location (no geolocation is ever requested).
export function useLocationStream(): { locations: FriendLocation[]; me: MyPosition | null } {
  return { locations: useFriendStream().locations, me: null };
}

import { FriendLocation, useFriendStream } from './friendStream';

export type { FriendLocation };

// The web build is view only: it shows friends' positions and never sends
// this browser's location (no geolocation is ever requested).
export function useLocationStream(): FriendLocation[] {
  return useFriendStream().locations;
}

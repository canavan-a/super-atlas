import { isStale, timeAgo } from '../time';
import type { Friend } from './useFriends';

// Slower than this (about 5 mph) counts as stopped: GPS jitter and walking
// around a shop shouldn't read as on the move.
const MOVING_MPS = 2.2;

const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Meters a second as whole miles an hour. */
export function mph(mps: number): number {
  return Math.round(mps * 2.23694);
}

/** Whether a friend's last fix says they're on the move. */
export function isMoving(w: { speedMps?: number; recordedAt?: string }, now = Date.now()): boolean {
  return w.speedMps !== undefined && w.speedMps >= MOVING_MPS && !!w.recordedAt && !isStale(w.recordedAt, now);
}

/** "23 min", "2 h 5 min", "3 days": how long ago iso was. */
export function durationSince(iso: string, now = Date.now()): string {
  const diff = Math.max(0, now - new Date(iso).getTime());
  if (diff < HOUR) return `${Math.max(1, Math.floor(diff / MINUTE))} min`;
  if (diff < DAY) {
    const min = Math.floor((diff % HOUR) / MINUTE);
    return `${Math.floor(diff / HOUR)} h${min ? ` ${min} min` : ''}`;
  }
  const days = Math.floor(diff / DAY);
  return days === 1 ? '1 day' : `${days} days`;
}

/**
 * One line on where a friend is at: "Moving · 42 mph", "Here for 23 min",
 * "Here for 3 h · seen 2 h ago" when their phone hasn't checked in lately,
 * or just "Seen 5 min ago". With a place name for where they've stopped
 * (placeName.ts), that stands in for "Here for …": "Stirrup Drive area",
 * "Starbucks · seen 2 h ago".
 */
export function whereabouts(
  w: { recordedAt: string; speedMps?: number; hereSince?: string },
  now = Date.now(),
  place?: string | null
): string {
  if (isMoving(w, now)) return `Moving · ${mph(w.speedMps!)} mph`;
  if (place) return isStale(w.recordedAt, now) ? `${place} · seen ${timeAgo(w.recordedAt, now)}` : place;
  if (w.hereSince) {
    const here = now - new Date(w.hereSince).getTime() < MINUTE ? 'Just got here' : `Here for ${durationSince(w.hereSince, now)}`;
    return isStale(w.recordedAt, now) ? `${here} · seen ${timeAgo(w.recordedAt, now)}` : here;
  }
  return `Seen ${timeAgo(w.recordedAt, now)}`;
}

/** Where to look up a place name for a friend (placeName.ts), if anywhere:
 *  only once they've stopped, since moving shows their speed instead. */
export function friendSpot(f: Friend, now = Date.now()): { latitude: number; longitude: number } | null {
  if (f.status !== 'accepted' || f.ghostMode || !f.lastSeenAt) return null;
  if (f.latitude === undefined || f.longitude === undefined) return null;
  if (isMoving({ speedMps: f.speedMps, recordedAt: f.lastSeenAt }, now)) return null;
  return { latitude: f.latitude, longitude: f.longitude };
}

/** The status line for a friend in the friends list, with the place name
 *  for where they are when there is one (friendSpot). */
export function friendStatus(f: Friend, now = Date.now(), place?: string | null): string {
  if (f.status === 'pending') return f.isIncomingRequest ? 'Wants to be friends' : 'Request sent';
  if (f.ghostMode) return 'In ghost mode';
  return f.lastSeenAt
    ? whereabouts({ recordedAt: f.lastSeenAt, speedMps: f.speedMps, hereSince: f.hereSince }, now, place)
    : 'No location yet';
}

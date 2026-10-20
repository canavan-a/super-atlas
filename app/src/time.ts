const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

// Positions older than this are drawn faded on the map.
export const STALE_AFTER_MS = HOUR;

export function isStale(iso: string, now = Date.now()): boolean {
  return now - new Date(iso).getTime() > STALE_AFTER_MS;
}

// "just now", "5 min ago", "3 h ago", "yesterday", "4 days ago", or a date.
export function timeAgo(iso: string, now = Date.now()): string {
  const diff = Math.max(0, now - new Date(iso).getTime());
  if (diff < MINUTE) return 'just now';
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)} min ago`;
  if (diff < DAY) return `${Math.floor(diff / HOUR)} h ago`;
  if (diff < 2 * DAY) return 'yesterday';
  if (diff < 7 * DAY) return `${Math.floor(diff / DAY)} days ago`;
  return new Date(iso).toLocaleDateString();
}

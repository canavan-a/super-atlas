import { useEffect, useMemo, useState } from 'react';
import { useFriends } from '../friends/useFriends';
import { useLocationStream } from '../location/useLocationStream';

// What both map screens (Android and web) show: friends' live positions,
// filtered to current friends (the stream only ever adds people, so this
// drops anyone just removed), your own position (Android only; null on the
// web), the count of requests waiting on you, which
// friend is tapped, and a clock that ticks each minute so "last seen" and
// faded markers stay current.
export function useMapData() {
  const { locations: streamed, me } = useLocationStream();
  const { friends } = useFriends();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60 * 1000);
    return () => clearInterval(t);
  }, []);

  const acceptedIds = useMemo(
    () => (friends ? new Set(friends.filter((f) => f.status === 'accepted').map((f) => f.friendId)) : null),
    [friends]
  );
  const locations = acceptedIds ? streamed.filter((l) => acceptedIds.has(l.userId)) : streamed;

  return {
    locations,
    me,
    now,
    pendingCount: friends?.filter((f) => f.isIncomingRequest).length ?? 0,
    noFriends: friends !== null && friends.length === 0,
    selected: locations.find((l) => l.userId === selectedId) ?? null,
    setSelectedId,
  };
}

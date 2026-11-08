import { useNavigation, useRoute } from '@react-navigation/native';
import { useEffect, useMemo, useState } from 'react';
import { useFriends } from '../friends/useFriends';
import { useLocationStream } from '../location/useLocationStream';

// What both map screens (Android and web) show: friends' live positions,
// filtered to current friends (the stream only ever adds people, so this
// drops anyone just removed), your own position (this phone's GPS; on the
// web, your phone's last report), the count of requests waiting on you, who is tapped (a friend's
// userId, or 'me' for you), a clock that ticks each minute so "last seen" and
// faded markers stay current, and focus: a friend the map was opened to show
// ("Show on map" in the friends list), once they're on it. The screen
// follows them and calls focusDone.
export function useMapData() {
  const { locations: streamed, me } = useLocationStream();
  const { friends } = useFriends();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const navigation = useNavigation<any>();
  const focusId: string | undefined = (useRoute<any>().params as any)?.focusId;

  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60 * 1000);
    return () => clearInterval(t);
  }, []);

  // Friends you may see: accepted and not in ghost mode. The server already
  // holds ghosts back; this also drops anyone the stream showed before they
  // went dark.
  const acceptedIds = useMemo(
    () =>
      friends
        ? new Set(friends.filter((f) => f.status === 'accepted' && !f.ghostMode).map((f) => f.friendId))
        : null,
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
    meSelected: selectedId === 'me' && me !== null,
    setSelectedId,
    focus: focusId ? (locations.find((l) => l.userId === focusId) ?? null) : null,
    focusDone: () => navigation.setParams({ focusId: undefined }),
  };
}

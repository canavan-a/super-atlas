import { useFocusEffect } from '@react-navigation/native';
import { useCallback, useState } from 'react';
import { apiJson } from '../api/client';

export type Friend = {
  friendshipId: string;
  friendId: string;
  displayName: string;
  avatarId?: string;
  status: 'pending' | 'accepted';
  isIncomingRequest: boolean;
  // Where they are, roughly: only for friends you can see (accepted, not in
  // ghost mode).
  lastSeenAt?: string;
  latitude?: number;
  longitude?: number;
  speedMps?: number;
  hereSince?: string;
  // They hide from everyone (only said of accepted friends).
  ghostMode: boolean;
};

// The friends list, reloaded whenever the screen using it comes into focus
// and, with pollMs, that often while it stays in focus (so speeds and new
// requests show up without leaving the screen).
export function useFriends(pollMs?: number) {
  const [friends, setFriends] = useState<Friend[] | null>(null);

  const reload = useCallback(async () => {
    try {
      setFriends(await apiJson<Friend[]>('/friends/'));
    } catch {
      // keep whatever we had; the next focus tries again
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      reload();
      if (!pollMs) return;
      const t = setInterval(reload, pollMs);
      return () => clearInterval(t);
    }, [reload, pollMs])
  );

  return { friends, reload, setFriends };
}

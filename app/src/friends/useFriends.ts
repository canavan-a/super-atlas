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
  lastSeenAt?: string;
};

// The friends list, reloaded whenever the screen using it comes into focus.
export function useFriends() {
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
    }, [reload])
  );

  return { friends, reload };
}

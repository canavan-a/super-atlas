import { StyleProp, Text, TextStyle } from 'react-native';
import { friendSpot, friendStatus } from '../friends/status';
import type { Friend } from '../friends/useFriends';
import { usePlaceName } from '../location/placeName';

// A friend's status line (friendStatus), naming the place they've stopped at
// when the phone can find one, as their card on the map does.
export function FriendStatusText({
  friend,
  now,
  style,
}: {
  friend: Friend;
  now: number;
  style?: StyleProp<TextStyle>;
}) {
  const place = usePlaceName(friendSpot(friend, now));
  return <Text style={style}>{friendStatus(friend, now, place)}</Text>;
}

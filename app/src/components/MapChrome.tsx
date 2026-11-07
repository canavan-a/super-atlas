import { useNavigation } from '@react-navigation/native';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuth } from '../auth/AuthContext';
import { isMoving, whereabouts } from '../friends/status';
import type { FriendLocation } from '../location/friendStream';
import { usePlaceName } from '../location/placeName';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { Avatar } from './Avatar';
import { BackgroundLocationBanner } from './BackgroundLocationBanner';
import { Icon } from './Icon';

// Everything drawn over the map, shared by the Android map (MapScreen.tsx)
// and the web one (MapScreen.web.tsx): the Friends button with its badge for
// requests waiting on you, the "fit everyone" / "show me" button, the card
// for a tapped friend (their picture and where they're at, by place name when
// the phone can find one; tap it for their own screen) or for tapping
// yourself ("You"; tap it for Settings), and the empty-state nudge. `notice`
// is a fixed line at the top (the web build's "view only"); without one, the
// top holds BackgroundLocationBanner when it applies. `group` is a tapped group
// marker whose people are too close together to zoom apart (map/cluster.ts):
// a card listing them, each tappable as if their own marker had been.
export type GroupPerson = { id: string; name: string; avatarId?: string; stale: boolean; isMe: boolean };

export function MapChrome({
  pendingCount,
  onFit,
  onLocate,
  selected,
  meSelected = false,
  me,
  now,
  noFriends,
  notice,
  group,
  onPick,
}: {
  pendingCount: number;
  onFit: () => void;
  /** Set right after "fit everyone": the button becomes "Show me" and calls
   *  this instead, so it toggles between everyone and you (Android only). */
  onLocate?: () => void;
  selected: FriendLocation | null;
  /** You were tapped: show the "You" card (Android only; the web has no you). */
  meSelected?: boolean;
  /** Your position, named on the "You" card. */
  me?: { latitude: number; longitude: number } | null;
  now: number;
  noFriends: boolean;
  notice?: string;
  group?: GroupPerson[] | null;
  /** Someone in `group` was tapped: their id ('me' for you). */
  onPick?: (id: string) => void;
}) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();
  const { user } = useAuth();
  // Where the tapped friend has stopped, by name; not looked up while they're
  // moving, since the card shows their speed then.
  const place = usePlaceName(selected && !isMoving(selected, now) ? selected : null);
  // And where you are, for the "You" card.
  const myPlace = usePlaceName(meSelected ? (me ?? null) : null);

  return (
    <>
      {notice ? (
        <View style={[styles.notice, { top: insets.top + 12 }]}>
          <Text style={styles.noticeText}>{notice}</Text>
        </View>
      ) : (
        <BackgroundLocationBanner style={{ ...styles.banner, top: insets.top + 12 }} />
      )}

      <View style={[styles.topRight, { top: insets.top + 12 }]}>
        <Pressable
          style={styles.fab}
          onPress={() => navigation.navigate('Friends')}
          accessibilityLabel={pendingCount ? `Friends, ${pendingCount} requests waiting` : 'Friends'}
        >
          <Icon name="friends" color={theme.text} size={22} />
          {pendingCount > 0 && (
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{pendingCount}</Text>
            </View>
          )}
        </Pressable>
        {onLocate ? (
          <Pressable style={[styles.fab, styles.fabGap]} onPress={onLocate} accessibilityLabel="Show me">
            <Icon name="locate" color={theme.text} size={22} />
          </Pressable>
        ) : (
          <Pressable style={[styles.fab, styles.fabGap]} onPress={onFit} accessibilityLabel="Show everyone">
            <Icon name="fit" color={theme.text} size={22} />
          </Pressable>
        )}
      </View>

      {meSelected && user && (
        <Pressable
          style={[styles.card, styles.cardRow, { bottom: insets.bottom + 24 }]}
          onPress={() => navigation.navigate('Settings')}
        >
          <Avatar name={user.displayName} avatarId={user.avatarId} size={44} />
          <View style={styles.flex}>
            <Text style={styles.cardName}>You</Text>
            {myPlace && <Text style={styles.cardMeta}>{myPlace}</Text>}
          </View>
          <Icon name="chevron" color={theme.textMuted} size={18} />
        </Pressable>
      )}

      {selected && (
        <Pressable
          style={[styles.card, styles.cardRow, { bottom: insets.bottom + 24 }]}
          onPress={() => navigation.navigate('Friend', { friendId: selected.userId })}
        >
          <Avatar
            name={selected.displayName}
            avatarId={selected.avatarId}
            size={44}
            stale={isStale(selected.recordedAt, now)}
          />
          <View style={styles.flex}>
            <Text style={styles.cardName}>{selected.displayName || 'Friend'}</Text>
            {place && <Text style={styles.cardMeta}>{place}</Text>}
            <Text style={styles.cardMeta}>{whereabouts(selected, now)}</Text>
          </View>
          <Icon name="chevron" color={theme.textMuted} size={18} />
        </Pressable>
      )}

      {group && group.length > 1 && !selected && !meSelected && (
        <View style={[styles.card, { bottom: insets.bottom + 24 }]}>
          <Text style={styles.cardName}>{group.length} people here</Text>
          {group.map((p) => (
            <Pressable key={p.id} style={[styles.cardRow, styles.groupRow]} onPress={() => onPick?.(p.id)}>
              <Avatar name={p.name} avatarId={p.avatarId} size={36} stale={p.stale} />
              <Text style={[styles.flex, styles.groupName]}>{p.isMe ? 'You' : p.name || 'Friend'}</Text>
              <Icon name="chevron" color={theme.textMuted} size={18} />
            </Pressable>
          ))}
        </View>
      )}

      {noFriends && !selected && !meSelected && !(group && group.length > 1) && (
        <Pressable style={[styles.card, { bottom: insets.bottom + 24 }]} onPress={() => navigation.navigate('Friends')}>
          <Text style={styles.cardName}>No friends yet</Text>
          <Text style={styles.cardMeta}>Tap to invite someone. You'll see each other here.</Text>
        </Pressable>
      )}
    </>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    topRight: { position: 'absolute', right: 16 },
    fab: {
      width: 48,
      height: 48,
      borderRadius: 24,
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 3 },
      shadowOpacity: 0.25,
      shadowRadius: 8,
      elevation: 5,
    },
    fabGap: { marginTop: 12 },
    badge: {
      position: 'absolute',
      top: -4,
      right: -4,
      minWidth: 20,
      height: 20,
      borderRadius: 10,
      paddingHorizontal: 5,
      backgroundColor: theme.danger,
      alignItems: 'center',
      justifyContent: 'center',
    },
    badgeText: { color: '#fff', fontSize: 12, fontWeight: '700' },
    notice: {
      position: 'absolute',
      left: 16,
      right: 80,
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 12,
      paddingVertical: 10,
      paddingHorizontal: 14,
    },
    noticeText: { color: theme.text, fontSize: 14 },
    banner: { position: 'absolute', left: 16, right: 80 },
    card: {
      position: 'absolute',
      left: 16,
      right: 16,
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 16,
      padding: 16,
    },
    cardRow: { flexDirection: 'row', alignItems: 'center', gap: 12 },
    flex: { flex: 1 },
    cardName: { color: theme.text, fontSize: 17, fontWeight: '600' },
    cardMeta: { color: theme.textMuted, marginTop: 4 },
    groupRow: { marginTop: 12 },
    groupName: { color: theme.text, fontSize: 16 },
  });

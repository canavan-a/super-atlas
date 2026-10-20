import { useNavigation } from '@react-navigation/native';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import type { FriendLocation } from '../location/friendStream';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { timeAgo } from '../time';
import { Icon } from './Icon';

export function initialsFor(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  const first = parts[0][0] ?? '';
  const last = parts.length > 1 ? parts[parts.length - 1][0] : '';
  return (first + last).toUpperCase();
}

// Everything drawn over the map, shared by the Android map (MapScreen.tsx)
// and the web one (MapScreen.web.tsx): the Friends button with its badge for
// requests waiting on you, the "fit everyone" button, the card for a tapped
// friend, and the empty-state nudge. `notice` is a fixed line at the top
// (the web build's "view only").
export function MapChrome({
  pendingCount,
  onFit,
  selected,
  now,
  noFriends,
  notice,
}: {
  pendingCount: number;
  onFit: () => void;
  selected: FriendLocation | null;
  now: number;
  noFriends: boolean;
  notice?: string;
}) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();

  return (
    <>
      {notice && (
        <View style={[styles.notice, { top: insets.top + 12 }]}>
          <Text style={styles.noticeText}>{notice}</Text>
        </View>
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
        <Pressable style={[styles.fab, styles.fabGap]} onPress={onFit} accessibilityLabel="Show everyone">
          <Icon name="fit" color={theme.text} size={22} />
        </Pressable>
      </View>

      {selected && (
        <View style={[styles.card, { bottom: insets.bottom + 24 }]}>
          <Text style={styles.cardName}>{selected.displayName || 'Friend'}</Text>
          <Text style={styles.cardMeta}>Seen {timeAgo(selected.recordedAt, now)}</Text>
        </View>
      )}

      {noFriends && !selected && (
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
    cardName: { color: theme.text, fontSize: 17, fontWeight: '600' },
    cardMeta: { color: theme.textMuted, marginTop: 4 },
  });

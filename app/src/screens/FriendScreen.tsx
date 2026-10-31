import { useNavigation, useRoute } from '@react-navigation/native';
import { useEffect, useState } from 'react';
import { Alert, ScrollView, StyleSheet, Text, View } from 'react-native';
import { apiJson } from '../api/client';
import { Avatar } from '../components/Avatar';
import { Icon } from '../components/Icon';
import { Button, Card } from '../components/ui';
import { durationSince, friendStatus, isMoving, mph } from '../friends/status';
import { useFriends } from '../friends/useFriends';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { timeAgo } from '../time';

// One friend, opened from the friends list or their card on the map: where
// they're at (moving and how fast, or how long they've been at this spot),
// whether they're in ghost mode, showing them on the map, and removing them.
// Refreshes every few seconds while open.
export default function FriendScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const navigation = useNavigation<any>();
  const { friendId } = useRoute<any>().params as { friendId: string };
  const { friends, reload } = useFriends(5000);
  const [error, setError] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const friend = friends?.find((f) => f.friendId === friendId);

  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 5000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (friend) navigation.setOptions({ title: friend.displayName });
  }, [navigation, friend?.displayName]); // eslint-disable-line react-hooks/exhaustive-deps

  // Gone from the list: removed (here or by them).
  useEffect(() => {
    if (friends && !friend) navigation.goBack();
  }, [friends, friend, navigation]);

  if (!friend) return <View style={styles.root} />;
  const name = friend.displayName || 'your friend';
  const accepted = friend.status === 'accepted';

  const act = async (path: string, method: string) => {
    setError(null);
    try {
      await apiJson(path, { method });
    } catch (e: any) {
      setError(e.message ?? 'Something went wrong');
    }
    await reload();
  };

  const remove = () => {
    const pending = friend.status === 'pending';
    Alert.alert(
      pending ? (friend.isIncomingRequest ? 'Decline request?' : 'Cancel request?') : `Remove ${name}?`,
      pending ? undefined : "You'll stop seeing each other on the map.",
      [
        { text: 'Keep', style: 'cancel' },
        {
          text: pending ? (friend.isIncomingRequest ? 'Decline' : 'Cancel request') : 'Remove',
          style: 'destructive',
          onPress: () => act(`/friends/${friend.friendshipId}`, 'DELETE'),
        },
      ]
    );
  };

  const visible = accepted && !friend.ghostMode && friend.lastSeenAt;
  const where = visible ? { recordedAt: friend.lastSeenAt!, speedMps: friend.speedMps, hereSince: friend.hereSince } : null;

  return (
    <ScrollView style={styles.root} contentContainerStyle={styles.content}>
      <View style={styles.top}>
        <Avatar name={friend.displayName} avatarId={friend.avatarId} size={88} />
        <Text style={styles.name}>{friend.displayName}</Text>
        <Text style={styles.status}>{friendStatus(friend, now)}</Text>
      </View>

      {where && (
        <Card style={styles.details}>
          <Row label="Speed" value={isMoving(where, now) ? `${mph(where.speedMps!)} mph` : 'Stopped'} styles={styles} />
          {where.hereSince && !isMoving(where, now) && (
            <Row label="At this spot for" value={durationSince(where.hereSince, now)} styles={styles} />
          )}
          <Row label="Last update" value={timeAgo(where.recordedAt, now)} styles={styles} last />
        </Card>
      )}

      {accepted && friend.ghostMode && (
        <Card style={styles.note}>
          <Icon name="ghost" color={theme.textMuted} size={22} />
          <Text style={[styles.noteText, styles.flex]}>
            {name} is in ghost mode, so nobody can see where they are.
          </Text>
        </Card>
      )}

      {visible && (
        <Button
          label="Show on map"
          onPress={() => navigation.navigate('Map', { focusId: friend.friendId })}
          style={styles.button}
        />
      )}

      {friend.isIncomingRequest && (
        <Button label="Accept" onPress={() => act(`/friends/${friend.friendshipId}/accept`, 'POST')} style={styles.button} />
      )}

      {error && <Text style={styles.error}>{error}</Text>}

      <Button
        label={
          friend.status === 'pending' ? (friend.isIncomingRequest ? 'Decline' : 'Cancel request') : `Remove ${name}`
        }
        onPress={remove}
        variant="danger"
        style={styles.button}
      />
    </ScrollView>
  );
}

function Row({
  label,
  value,
  styles,
  last,
}: {
  label: string;
  value: string;
  styles: ReturnType<typeof makeStyles>;
  last?: boolean;
}) {
  return (
    <View style={[styles.detailRow, !last && styles.detailDivider]}>
      <Text style={styles.detailLabel}>{label}</Text>
      <Text style={styles.detailValue}>{value}</Text>
    </View>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    content: { padding: 16, paddingBottom: 40 },
    flex: { flex: 1 },
    top: { alignItems: 'center', marginTop: 8, marginBottom: 20 },
    name: { color: theme.text, fontSize: 22, fontWeight: '700', marginTop: 12 },
    status: { color: theme.textMuted, fontSize: 15, marginTop: 4, textAlign: 'center' },
    details: { paddingVertical: 4, marginBottom: 12 },
    detailRow: { flexDirection: 'row', justifyContent: 'space-between', paddingVertical: 12 },
    detailDivider: { borderBottomWidth: StyleSheet.hairlineWidth, borderBottomColor: theme.border },
    detailLabel: { color: theme.textMuted, fontSize: 15 },
    detailValue: { color: theme.text, fontSize: 15, fontWeight: '600' },
    note: { flexDirection: 'row', alignItems: 'center', gap: 12, marginBottom: 12 },
    noteText: { color: theme.text, fontSize: 15 },
    button: { marginTop: 12 },
    error: { color: theme.danger, marginTop: 12, textAlign: 'center' },
  });

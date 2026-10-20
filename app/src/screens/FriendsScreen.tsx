import { useState } from 'react';
import { Alert, FlatList, Pressable, Share, StyleSheet, Switch, Text, View } from 'react-native';
import { apiJson } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { Icon } from '../components/Icon';
import { Button, Card, TextField } from '../components/ui';
import { Friend, useFriends } from '../friends/useFriends';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { timeAgo } from '../time';

export default function FriendsScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { ghostMode, setGhostMode } = useAuth();
  const { friends, reload } = useFriends();
  const [sharing, setSharing] = useState(false);
  const [showEmail, setShowEmail] = useState(false);
  const [email, setEmail] = useState('');
  const [emailBusy, setEmailBusy] = useState(false);
  const [notice, setNotice] = useState<{ text: string; error?: boolean } | null>(null);

  const incoming = (friends ?? []).filter((f) => f.isIncomingRequest);
  const others = (friends ?? []).filter((f) => !f.isIncomingRequest);

  const invite = async () => {
    setSharing(true);
    try {
      const { url } = await apiJson<{ url: string }>('/friends/invite');
      await Share.share({ message: `Add me on Super Atlas so we can see each other on the map: ${url}` });
    } catch (e: any) {
      setNotice({ text: e.message ?? 'Could not load your invite link', error: true });
    } finally {
      setSharing(false);
    }
  };

  const addByEmail = async () => {
    setEmailBusy(true);
    setNotice(null);
    try {
      const res = await apiJson<{ status?: string; larp?: boolean }>('/friends/', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim().toLowerCase() }),
      });
      setEmail('');
      setShowEmail(false);
      setNotice({ text: res.status === 'accepted' ? "You're now friends." : 'Request sent.' });
      await reload();
    } catch (e: any) {
      setNotice({ text: e.message ?? 'Could not send request', error: true });
    } finally {
      setEmailBusy(false);
    }
  };

  const accept = async (f: Friend) => {
    await apiJson(`/friends/${f.friendshipId}/accept`, { method: 'POST' }).catch(() => {});
    await reload();
  };

  const remove = async (f: Friend) => {
    await apiJson(`/friends/${f.friendshipId}`, { method: 'DELETE' }).catch(() => {});
    await reload();
  };

  const confirmRemove = (f: Friend) => {
    const pending = f.status === 'pending';
    Alert.alert(
      pending ? 'Cancel request?' : `Remove ${f.displayName}?`,
      pending ? undefined : "You'll stop seeing each other on the map.",
      [
        { text: 'Keep', style: 'cancel' },
        { text: pending ? 'Cancel request' : 'Remove', style: 'destructive', onPress: () => remove(f) },
      ]
    );
  };

  const toggleGhost = (on: boolean) => {
    setGhostMode(on).catch(() => setNotice({ text: 'Could not change ghost mode', error: true }));
  };

  const header = (
    <View>
      <Card style={styles.ghostCard}>
        <Icon name="ghost" color={ghostMode ? theme.primary : theme.textMuted} size={22} />
        <View style={styles.ghostText}>
          <Text style={styles.rowTitle}>Ghost mode</Text>
          <Text style={styles.rowMeta}>{ghostMode ? 'Your friends can’t see you.' : 'Hide your location from everyone.'}</Text>
        </View>
        <Switch
          value={ghostMode}
          onValueChange={toggleGhost}
          trackColor={{ true: theme.primary, false: theme.border }}
          thumbColor={theme.surfaceAlt}
        />
      </Card>

      <Button label="Invite a friend" onPress={invite} loading={sharing} style={styles.inviteButton} />

      {showEmail ? (
        <View style={styles.emailRow}>
          <TextField
            placeholder="Friend's email"
            autoCapitalize="none"
            keyboardType="email-address"
            autoFocus
            value={email}
            onChangeText={setEmail}
            onSubmitEditing={addByEmail}
            style={styles.emailInput}
          />
          <Button label="Add" onPress={addByEmail} loading={emailBusy} disabled={!email.trim()} variant="secondary" />
        </View>
      ) : (
        <Pressable onPress={() => setShowEmail(true)} style={styles.emailLink} hitSlop={8}>
          <Text style={styles.link}>Add by email instead</Text>
        </Pressable>
      )}

      {notice && <Text style={[styles.notice, notice.error && styles.noticeError]}>{notice.text}</Text>}

      {incoming.length > 0 && (
        <>
          <Text style={styles.section}>Requests</Text>
          {incoming.map((f) => (
            <Card key={f.friendshipId} style={styles.row}>
              <Text style={[styles.rowTitle, styles.flex]}>{f.displayName}</Text>
              <Button label="Accept" onPress={() => accept(f)} style={styles.smallButton} />
              <Button label="Decline" onPress={() => remove(f)} variant="secondary" style={styles.smallButton} />
            </Card>
          ))}
        </>
      )}

      {others.length > 0 && <Text style={styles.section}>Friends</Text>}
    </View>
  );

  return (
    <FlatList
      style={styles.root}
      contentContainerStyle={styles.content}
      data={others}
      keyExtractor={(f) => f.friendshipId}
      ListHeaderComponent={header}
      renderItem={({ item }) => (
        <Pressable onPress={() => confirmRemove(item)}>
          <Card style={styles.row}>
            <View style={styles.flex}>
              <Text style={styles.rowTitle}>{item.displayName}</Text>
              <Text style={styles.rowMeta}>
                {item.status === 'pending'
                  ? 'Request sent'
                  : item.lastSeenAt
                    ? `Seen ${timeAgo(item.lastSeenAt)}`
                    : 'Location hidden'}
              </Text>
            </View>
          </Card>
        </Pressable>
      )}
      ListEmptyComponent={
        friends && incoming.length === 0 ? (
          <Text style={styles.empty}>No friends yet. Send someone your invite link to get started.</Text>
        ) : null
      }
    />
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    content: { padding: 16, paddingBottom: 40 },
    flex: { flex: 1 },
    ghostCard: { flexDirection: 'row', alignItems: 'center', gap: 12 },
    ghostText: { flex: 1 },
    inviteButton: { marginTop: 16 },
    emailLink: { alignSelf: 'center', marginTop: 14 },
    link: { color: theme.primary, fontWeight: '600' },
    emailRow: { flexDirection: 'row', gap: 8, marginTop: 12 },
    emailInput: { flex: 1 },
    notice: { color: theme.textMuted, marginTop: 12, textAlign: 'center' },
    noticeError: { color: theme.danger },
    section: {
      color: theme.textMuted,
      fontSize: 13,
      fontWeight: '600',
      textTransform: 'uppercase',
      marginTop: 24,
      marginBottom: 8,
    },
    row: { flexDirection: 'row', alignItems: 'center', gap: 8, marginBottom: 8, paddingVertical: 14 },
    rowTitle: { color: theme.text, fontSize: 16, fontWeight: '600' },
    rowMeta: { color: theme.textMuted, fontSize: 13, marginTop: 2 },
    smallButton: { paddingVertical: 8, paddingHorizontal: 12 },
    empty: { color: theme.textMuted, textAlign: 'center', marginTop: 32, lineHeight: 20 },
  });

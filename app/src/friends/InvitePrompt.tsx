import * as Linking from 'expo-linking';
import { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { apiJson } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { Overlay } from '../components/Overlay';
import { Button } from '../components/ui';
import { Theme, useThemedStyles } from '../theme';

// Pulls the invite code out of superatlas://invite/<code> or
// https://<server>/i/<code>.
export function inviteCodeFromURL(url: string | null): string | null {
  if (!url) return null;
  const match = url.match(/(?:\/\/invite\/|\/i\/)([a-z0-9]+)/i);
  return match ? match[1].toLowerCase() : null;
}

// Asks "Add <name> as a friend?" when the app is opened from an invite link.
// If you aren't signed in yet, the code waits in memory until you are.
export function InvitePrompt() {
  const styles = useThemedStyles(makeStyles);
  const url = Linking.useLinkingURL();
  const { status } = useAuth();
  const [code, setCode] = useState<string | null>(null);
  const [name, setName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);

  useEffect(() => {
    const next = inviteCodeFromURL(url);
    if (next) {
      setCode(next);
      setName(null);
      setError(null);
      setDone(false);
    }
  }, [url]);

  useEffect(() => {
    if (!code || status !== 'signedIn' || name || error) return;
    apiJson<{ displayName: string }>(`/invites/${code}`)
      .then((res) => setName(res.displayName))
      .catch((e) => setError(e.message ?? 'This invite link doesn’t work anymore'));
  }, [code, status, name, error]);

  const close = () => {
    setCode(null);
    setName(null);
    setError(null);
    setDone(false);
  };

  const accept = async () => {
    setBusy(true);
    try {
      await apiJson(`/invites/${code}/accept`, { method: 'POST' });
      setDone(true);
    } catch (e: any) {
      setError(e.message ?? 'Could not add friend');
    } finally {
      setBusy(false);
    }
  };

  const visible = status === 'signedIn' && !!code && (!!name || !!error);

  return (
    <Overlay visible={visible} onClose={close} style={styles.center}>
      <View style={styles.card}>
        {error ? (
          <>
            <Text style={styles.title}>Invite link</Text>
            <Text style={styles.body}>{error}</Text>
            <Button label="OK" onPress={close} variant="secondary" />
          </>
        ) : done ? (
          <>
            <Text style={styles.title}>You and {name} are friends</Text>
            <Text style={styles.body}>You'll see each other on the map.</Text>
            <Button label="Done" onPress={close} />
          </>
        ) : (
          <>
            <Text style={styles.title}>Add {name} as a friend?</Text>
            <Text style={styles.body}>You'll see each other on the map. Either of you can remove the other any time.</Text>
            <Button label="Add friend" onPress={accept} loading={busy} />
            <Button label="Not now" onPress={close} variant="secondary" style={styles.gap} />
          </>
        )}
      </View>
    </Overlay>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    center: { justifyContent: 'center', padding: 24 },
    card: {
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 16,
      padding: 20,
    },
    title: { color: theme.text, fontSize: 19, fontWeight: '700', marginBottom: 8 },
    body: { color: theme.textMuted, fontSize: 15, lineHeight: 21, marginBottom: 20 },
    gap: { marginTop: 10 },
  });

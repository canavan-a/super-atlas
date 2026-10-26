import { useNavigation } from '@react-navigation/native';
import { useEffect, useState } from 'react';
import { Alert, Platform, Pressable, ScrollView, StyleSheet, Switch, Text, View } from 'react-native';
import {
  Release,
  compareVersions,
  fetchLatestRelease,
  getVersionInfo,
  installRelease,
  onDownloadProgress,
  updatesSupported,
} from '../appUpdater';
import { useAuth } from '../auth/AuthContext';
import { Button, Card } from '../components/ui';
import {
  backgroundNotificationsEnabled,
  hasBackgroundNotificationSwitch,
  setBackgroundNotifications,
} from '../notifications';
import { THEMES, THEME_OPTIONS, Theme, useTheme, useThemeSetting, useThemedStyles } from '../theme';

// Asks before a destructive action: Alert on Android, the browser's own
// dialog on the web, where react-native-web's Alert does nothing.
function confirmThen(title: string, message: string, action: string, onConfirm: () => void) {
  if (Platform.OS === 'web') {
    if (window.confirm(`${title}\n\n${message}`)) onConfirm();
    return;
  }
  Alert.alert(title, message, [
    { text: 'Cancel', style: 'cancel' },
    { text: action, style: 'destructive', onPress: onConfirm },
  ]);
}

type UpdateState =
  | { kind: 'idle' }
  | { kind: 'checking' }
  | { kind: 'upToDate' }
  | { kind: 'available'; release: Release }
  | { kind: 'downloading'; release: Release; percent: number | null }
  | { kind: 'error'; message: string };

// Deliberately short: theme, notifications (sideload), updates (sideload),
// sign out, deleting the account, and the privacy policy.
export default function SettingsScreen() {
  const styles = useThemedStyles(makeStyles);
  const { themeName, setThemeName } = useThemeSetting();
  const { user, logout, deleteAccount } = useAuth();
  const [version, setVersion] = useState<string | null>(null);
  const [update, setUpdate] = useState<UpdateState>({ kind: 'idle' });
  const theme = useTheme();
  const [bgNotify, setBgNotify] = useState(false);
  const navigation = useNavigation<any>();
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (hasBackgroundNotificationSwitch) backgroundNotificationsEnabled().then(setBgNotify);
  }, []);

  const toggleBgNotify = async (on: boolean) => {
    setBgNotify(on);
    try {
      const result = await setBackgroundNotifications(on);
      setBgNotify(result);
      if (on && !result) Alert.alert('Notifications are off', 'Allow notifications for Super Atlas in Android settings first.');
    } catch {
      setBgNotify(!on);
    }
  };

  useEffect(() => {
    if (!updatesSupported) return;
    getVersionInfo()
      .then((v) => setVersion(v.versionName))
      .catch(() => {});
  }, []);

  const checkForUpdates = async () => {
    setUpdate({ kind: 'checking' });
    try {
      const latest = await fetchLatestRelease();
      if (latest && version && compareVersions(latest.version, version) > 0) {
        setUpdate({ kind: 'available', release: latest });
      } else {
        setUpdate({ kind: 'upToDate' });
      }
    } catch (e: any) {
      setUpdate({ kind: 'error', message: e.message ?? 'Could not check for updates' });
    }
  };

  const installUpdate = async (release: Release) => {
    setUpdate({ kind: 'downloading', release, percent: null });
    const stop = onDownloadProgress((p) => {
      if (p.bytesTotal > 0) {
        setUpdate({ kind: 'downloading', release, percent: Math.round((100 * p.bytesDownloaded) / p.bytesTotal) });
      }
    });
    try {
      await installRelease(release);
      setUpdate({ kind: 'available', release });
    } catch (e: any) {
      setUpdate({ kind: 'error', message: e.message ?? 'Update failed' });
    } finally {
      stop();
    }
  };

  const confirmSignOut = () =>
    confirmThen('Sign out?', 'Your friends will stop seeing you until you sign back in.', 'Sign out', () => logout());

  const confirmDelete = () =>
    confirmThen(
      'Delete your account?',
      "This permanently deletes your account, your friends list and your location from the server, and signs you out everywhere. It can't be undone.",
      'Delete',
      async () => {
        setDeleting(true);
        setDeleteError(null);
        try {
          await deleteAccount();
        } catch (e: any) {
          setDeleteError(e.message ?? 'Could not delete your account');
          setDeleting(false);
        }
      }
    );

  return (
    <ScrollView style={styles.root} contentContainerStyle={styles.content}>
      <Text style={styles.section}>Theme</Text>
      <View style={styles.themeRow}>
        {THEME_OPTIONS.map((opt) => (
          <Pressable
            key={opt.name}
            style={[styles.swatch, themeName === opt.name && styles.swatchActive]}
            onPress={() => setThemeName(opt.name)}
            accessibilityRole="radio"
            accessibilityState={{ selected: themeName === opt.name }}
          >
            <View style={styles.swatchColors}>
              <View style={[styles.swatchDot, { backgroundColor: THEMES[opt.name].bg }]} />
              <View style={[styles.swatchDot, { backgroundColor: THEMES[opt.name].primary }]} />
            </View>
            <Text style={styles.swatchLabel}>{opt.label}</Text>
          </Pressable>
        ))}
      </View>

      {hasBackgroundNotificationSwitch && (
        <>
          <Text style={styles.section}>Notifications</Text>
          <Card style={styles.row}>
            <View style={styles.rowText}>
              <Text style={styles.account}>Background notifications</Text>
              <Text style={styles.hint}>
                Get friend requests while the app is closed. Keeps a small connection open, shown as a quiet
                notification.
              </Text>
            </View>
            <Switch
              value={bgNotify}
              onValueChange={toggleBgNotify}
              trackColor={{ true: theme.primary, false: theme.border }}
              thumbColor={theme.surfaceAlt}
            />
          </Card>
        </>
      )}

      {updatesSupported && (
        <>
          <Text style={styles.section}>Updates</Text>
          <Text style={styles.account}>
            Version {version ?? '…'}
            {update.kind === 'upToDate' && ' · up to date'}
          </Text>
          {update.kind === 'available' ? (
            <Button
              label={`Install ${update.release.version}`}
              onPress={() => installUpdate(update.release)}
              style={styles.buttonGap}
            />
          ) : update.kind === 'downloading' ? (
            <Button
              label={update.percent === null ? 'Downloading…' : `Downloading ${update.percent}%`}
              onPress={() => {}}
              disabled
              style={styles.buttonGap}
            />
          ) : (
            <Button
              label="Check for updates"
              onPress={checkForUpdates}
              loading={update.kind === 'checking'}
              variant="secondary"
              style={styles.buttonGap}
            />
          )}
          {update.kind === 'error' && <Text style={styles.error}>{update.message}</Text>}
        </>
      )}

      <Text style={styles.section}>Account</Text>
      {user && (
        <Text style={styles.account}>
          {user.displayName} · {user.email}
        </Text>
      )}
      <Button label="Sign out" onPress={confirmSignOut} variant="danger" style={styles.buttonGap} />
      <Button
        label="Delete account"
        onPress={confirmDelete}
        loading={deleting}
        variant="secondary"
        style={styles.buttonGap}
      />
      {deleteError && <Text style={styles.error}>{deleteError}</Text>}

      <Pressable onPress={() => navigation.navigate('Privacy')} hitSlop={8} accessibilityRole="link" style={styles.legal}>
        <Text style={styles.legalText}>Privacy policy</Text>
      </Pressable>
    </ScrollView>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    content: { padding: 16, paddingBottom: 40 },
    section: {
      color: theme.textMuted,
      fontSize: 13,
      fontWeight: '600',
      textTransform: 'uppercase',
      marginTop: 16,
      marginBottom: 10,
    },
    themeRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 10 },
    swatch: {
      alignItems: 'center',
      gap: 6,
      padding: 10,
      borderRadius: 10,
      borderWidth: 2,
      borderColor: theme.border,
      backgroundColor: theme.surface,
    },
    swatchActive: { borderColor: theme.primary },
    swatchColors: {
      flexDirection: 'row',
      borderRadius: 14,
      overflow: 'hidden',
      borderWidth: 1,
      borderColor: theme.border,
    },
    swatchDot: { width: 20, height: 28 },
    swatchLabel: { fontSize: 12, fontWeight: '600', color: theme.text },
    account: { color: theme.text, fontSize: 15 },
    buttonGap: { marginTop: 16 },
    error: { color: theme.danger, marginTop: 10 },
    row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
    rowText: { flex: 1 },
    hint: { color: theme.textMuted, fontSize: 13, marginTop: 2, lineHeight: 18 },
    legal: { alignSelf: 'center', paddingVertical: 10, marginTop: 24 },
    legalText: { color: theme.textMuted, fontSize: 12 },
  });

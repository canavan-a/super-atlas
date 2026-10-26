import { StyleSheet, Text, View } from 'react-native';
import { Button } from '../components/ui';
import { PermissionsState } from '../location/usePermissionsGate';
import { Theme, useThemedStyles } from '../theme';

// Blocks the app until location (foreground + background) is granted.
// Explains why before the OS prompt, which both Apple and Google expect for
// background location. Once the OS stops showing its prompt, the button
// opens Settings instead, and the screen re-checks when you come back.
export default function PermissionsGateScreen({ permissions }: { permissions: PermissionsState }) {
  const styles = useThemedStyles(makeStyles);
  const needsBackground = permissions.foregroundLocationGranted && !permissions.backgroundLocationGranted;

  return (
    <View style={styles.root}>
      <Text style={styles.title}>Share your location</Text>
      {/* Google Play's "prominent disclosure" for background location: what is
          collected, that it continues when the app is closed or not in use,
          and what it's used for, shown before the OS prompt. Keep it in step
          with the privacy policy (src/privacy/policy.ts). */}
      <Text style={styles.body}>
        Super Atlas collects location data to show your location to your friends on a map, even when the app is
        closed or not in use.
      </Text>
      <Text style={styles.body}>
        Only people you accept as friends can see it, and only your latest position is kept. It is never sold or
        shared with anyone else. You can hide from everyone at any time with ghost mode.
      </Text>
      <Text style={styles.body}>
        {needsBackground
          ? 'One more step: choose "Allow all the time" so your friends still see you when the app is closed.'
          : 'Allow location access, then choose "Allow all the time" so your friends still see you when the app is closed.'}
      </Text>
      <Button
        label={permissions.mustOpenSettings ? 'Open Settings' : 'Continue'}
        onPress={permissions.requestAll}
        style={styles.button}
      />
      {permissions.mustOpenSettings && (
        <Text style={styles.hint}>In Settings, go to Location and pick "Allow all the time".</Text>
      )}
    </View>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg, padding: 24, justifyContent: 'center' },
    title: { color: theme.text, fontSize: 26, fontWeight: '700', marginBottom: 12 },
    body: { color: theme.textMuted, fontSize: 15, lineHeight: 21, marginBottom: 12 },
    button: { marginTop: 16 },
    hint: { color: theme.textMuted, fontSize: 13, marginTop: 12, textAlign: 'center' },
  });

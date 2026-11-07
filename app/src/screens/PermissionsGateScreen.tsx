import { Platform, StyleSheet, Text, View } from 'react-native';
import { Button } from '../components/ui';
import { PermissionsState } from '../location/usePermissionsGate';
import { Theme, useThemedStyles } from '../theme';

// Explains why before the OS prompts, which both Apple and Google expect for
// background location. On Android it blocks the app until location is
// granted, and once the OS stops showing its prompt the button opens
// Settings instead (the screen re-checks when you come back). On iOS it's
// shown only until the prompt is answered, either way (usePermissionsGate);
// Apple's guidance for a screen like this is one button that leads to the
// system prompt ("Continue", never "Allow"), and no way around the prompt.
// Declining background location only brings up BackgroundLocationBanner.
export default function PermissionsGateScreen({ permissions }: { permissions: PermissionsState }) {
  const styles = useThemedStyles(makeStyles);
  const ios = Platform.OS === 'ios';

  return (
    <View style={styles.root}>
      <Text style={styles.title}>Share your location</Text>
      {/* Google Play's "prominent disclosure" for background location, and
          Apple's purpose strings (ios/SuperAtlasApp/Info.plist) in more
          words: what is collected, that it continues when the app is closed
          or not in use, who sees it and what it's used for, shown before the
          OS prompt. Keep it in step with the privacy policy
          (src/privacy/policy.ts). */}
      <Text style={styles.body}>
        Super Atlas collects location data to show your location to your friends on a map, even when the app is
        closed or not in use.
      </Text>
      <Text style={styles.body}>
        Only people you accept as friends can see it, and only your latest position is kept. It is never sold or
        shared with anyone else. You can hide from everyone at any time with ghost mode.
      </Text>
      {ios ? (
        <Text style={styles.body}>
          Allow location while using the app, then choose "Change to Always Allow" so your friends still see you when
          the app is closed. Keep Precise Location on, or friends only see roughly where you are. Allowing Motion &
          Fitness next lets Super Atlas notice when you've stopped, to save battery.
        </Text>
      ) : (
        <Text style={styles.body}>
          Allow location access, then choose "Allow all the time" so your friends still see you when the app is closed.
          Allowing physical activity next lets Super Atlas notice straight away when you start driving or cycling.
        </Text>
      )}
      <Button
        label={permissions.mustOpenSettings ? 'Open Settings' : 'Continue'}
        onPress={permissions.requestAll}
        style={styles.button}
      />
      {permissions.mustOpenSettings && (
        <Text style={styles.hint}>In Settings, go to Location and pick "Allow all the time" (or at least "While using the app").</Text>
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

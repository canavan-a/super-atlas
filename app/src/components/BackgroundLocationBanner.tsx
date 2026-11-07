import { Alert, Linking, Platform, Pressable, StyleSheet, Text, ViewStyle } from 'react-native';
import { usePermissions } from '../location/permissionsContext';
import { Theme, useThemedStyles } from '../theme';

// Shown on the main screens while friends can't see you properly:
//  - location off altogether (iOS only: Android's gate blocks the app until
//    it's allowed): the app is view-only; tapping opens Settings;
//  - not allowed "all the time" / "Always": friends only see you while it's
//    open. Tapping repeats Google Play's prominent disclosure (as on
//    PermissionsGateScreen) before the OS prompt, since someone may have
//    skipped it there;
//  - iOS with Precise Location off: friends see roughly where you are.
export function BackgroundLocationBanner({ style }: { style?: ViewStyle }) {
  const styles = useThemedStyles(makeStyles);
  const permissions = usePermissions();
  if (!permissions) return null;
  const ios = Platform.OS === 'ios';

  let title: string;
  let body: string;
  let onPress: () => void;
  if (!permissions.foregroundLocationGranted) {
    title = "You're not sharing your location";
    body = 'Friends can\'t see you. Tap to allow location in Settings.';
    onPress = () => permissions.requestBackground();
  } else if (!permissions.backgroundLocationGranted) {
    title = "Friends can't see you when the app is closed";
    body = ios ? 'Tap to allow location always.' : 'Tap to allow location all the time.';
    onPress = () =>
      Alert.alert(
        ios ? 'Allow location always' : 'Allow location all the time',
        'Super Atlas collects location data to show your location to your friends on a map, even when the app is ' +
          'closed or not in use. Only your friends can see it, and only your latest position is kept.\n\n' +
          (permissions.backgroundMustOpenSettings
            ? ios
              ? 'In Settings, go to Location and choose "Always".'
              : 'In Settings, go to Location and choose "Allow all the time".'
            : ios
              ? 'Choose "Change to Always Allow" on the next screen.'
              : 'Choose "Allow all the time" on the next screen.'),
        [
          { text: 'Not now', style: 'cancel' },
          { text: 'Continue', onPress: () => permissions.requestBackground() },
        ]
      );
  } else if (permissions.preciseLocationOff) {
    title = 'Friends only see roughly where you are';
    body = 'Tap to turn on Precise Location in Settings.';
    onPress = () => Linking.openSettings();
  } else {
    return null;
  }

  return (
    <Pressable style={[styles.banner, style]} onPress={onPress} accessibilityRole="button">
      <Text style={styles.title}>{title}</Text>
      <Text style={styles.body}>{body}</Text>
    </Pressable>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    banner: {
      backgroundColor: theme.dangerBg,
      borderColor: theme.danger,
      borderWidth: 1,
      borderRadius: 12,
      paddingVertical: 10,
      paddingHorizontal: 14,
    },
    title: { color: theme.danger, fontSize: 14, fontWeight: '600' },
    body: { color: theme.text, fontSize: 13, marginTop: 2 },
  });

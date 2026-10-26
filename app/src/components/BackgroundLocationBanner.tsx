import { Alert, Pressable, StyleSheet, Text, ViewStyle } from 'react-native';
import { usePermissions } from '../location/permissionsContext';
import { Theme, useThemedStyles } from '../theme';

// Shown on the main screens until location is allowed "all the time". The
// app works without it, but friends only see you while it's open. Tapping it
// repeats Google Play's prominent disclosure (as on PermissionsGateScreen)
// before the OS prompt, since someone may have skipped it there.
export function BackgroundLocationBanner({ style }: { style?: ViewStyle }) {
  const styles = useThemedStyles(makeStyles);
  const permissions = usePermissions();
  if (!permissions || permissions.backgroundLocationGranted) return null;

  const onPress = () =>
    Alert.alert(
      'Allow location all the time',
      'Super Atlas collects location data to show your location to your friends on a map, even when the app is ' +
        'closed or not in use. Only your friends can see it, and only your latest position is kept.\n\n' +
        'Choose "Allow all the time" on the next screen.',
      [
        { text: 'Not now', style: 'cancel' },
        { text: 'Continue', onPress: () => permissions.requestBackground() },
      ]
    );

  return (
    <Pressable style={[styles.banner, style]} onPress={onPress} accessibilityRole="button">
      <Text style={styles.title}>Friends can't see you when the app is closed</Text>
      <Text style={styles.body}>Tap to allow location all the time.</Text>
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

import { Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { Button } from '../components/ui';
import { THEMES, THEME_OPTIONS, Theme, useThemeSetting, useThemedStyles } from '../theme';

// Deliberately short: theme, and sign out.
export default function SettingsScreen() {
  const styles = useThemedStyles(makeStyles);
  const { themeName, setThemeName } = useThemeSetting();
  const { user, logout } = useAuth();

  const confirmSignOut = () =>
    Alert.alert('Sign out?', 'Your friends will stop seeing you until you sign back in.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Sign out', style: 'destructive', onPress: () => logout() },
    ]);

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

      <Text style={styles.section}>Account</Text>
      {user && (
        <Text style={styles.account}>
          {user.displayName} · {user.email}
        </Text>
      )}
      <Button label="Sign out" onPress={confirmSignOut} variant="danger" style={styles.signOut} />
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
    signOut: { marginTop: 16 },
  });

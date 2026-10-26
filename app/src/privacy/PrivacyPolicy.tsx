// The privacy policy (policy.ts) as a screen: in the app from Settings, and
// on the web as a page of its own at /privacy (index.web.tsx), as in
// super-chat.
import { Linking, Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { Theme, useThemedStyles } from '../theme';
import { CONTACT, POLICY, UPDATED, type PolicyBlock } from './policy';

type Styles = ReturnType<typeof makeStyles>;

/** Text with **bold** runs. */
function Rich({ text, styles }: { text: string; styles: Styles }) {
  return (
    <Text style={styles.p}>
      {text.split('**').map((part, i) =>
        i % 2 ? (
          <Text key={i} style={styles.bold}>
            {part}
          </Text>
        ) : (
          part
        )
      )}
    </Text>
  );
}

function Block({ b, styles }: { b: PolicyBlock; styles: Styles }) {
  if ('p' in b) return <Rich text={b.p} styles={styles} />;
  return (
    <View style={styles.list}>
      {b.list.map((item, i) => (
        <View key={i} style={styles.item}>
          <Text style={styles.bullet}>•</Text>
          <View style={styles.itemText}>
            <Rich text={item} styles={styles} />
          </View>
        </View>
      ))}
    </View>
  );
}

export default function PrivacyPolicy() {
  const styles = useThemedStyles(makeStyles);
  return (
    <ScrollView style={styles.root} contentContainerStyle={styles.content}>
      {/* In the app the screen header says "Privacy policy" already. */}
      {Platform.OS === 'web' && <Text style={styles.title}>Super Atlas Privacy Policy</Text>}
      <Text style={styles.updated}>Last updated {UPDATED}</Text>
      {POLICY.map((section) => (
        <View key={section.title} style={styles.section}>
          <Text style={styles.h2} accessibilityRole="header">
            {section.title}
          </Text>
          {section.blocks.map((b, i) => (
            <Block key={i} b={b} styles={styles} />
          ))}
        </View>
      ))}
      <Pressable onPress={() => Linking.openURL(`mailto:${CONTACT}`).catch(() => {})}>
        <Text style={styles.link}>{CONTACT}</Text>
      </Pressable>
    </ScrollView>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    content: { padding: 20, paddingBottom: 48, maxWidth: 720, width: '100%', alignSelf: 'center' },
    title: { fontSize: 24, fontWeight: '800', color: theme.text, marginBottom: 4 },
    updated: { fontSize: 12, color: theme.textMuted, marginBottom: 8 },
    section: { marginTop: 18, gap: 8 },
    h2: { fontSize: 17, fontWeight: '700', color: theme.text },
    p: { fontSize: 14, lineHeight: 21, color: theme.text },
    bold: { fontWeight: '700' },
    list: { gap: 6 },
    item: { flexDirection: 'row', gap: 8 },
    bullet: { fontSize: 14, lineHeight: 21, color: theme.textMuted },
    itemText: { flex: 1 },
    link: { marginTop: 20, fontSize: 14, color: theme.primary },
  });

import { useEffect, useState } from 'react';
import { Image, StyleSheet, Text, View, type StyleProp, type ViewStyle } from 'react-native';
import { API_BASE_URL } from '../api/client';
import { useTheme } from '../theme';

export function initialsFor(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  const first = parts[0][0] ?? '';
  const last = parts.length > 1 ? parts[parts.length - 1][0] : '';
  return (first + last).toUpperCase();
}

/** A profile picture's address (the server's /avatars/{id}). */
export function avatarUrl(avatarId: string): string {
  return `${API_BASE_URL}/avatars/${avatarId}`;
}

// Someone's round profile picture, or their initials ("AC" for Aidan
// Canavan) on the friend marker color when they have none or it won't load.
// The map's markers, the friends list and Settings all use it.
export function Avatar({
  name,
  avatarId,
  size,
  stale = false,
  style,
}: {
  name: string;
  avatarId?: string;
  size: number;
  /** Faded, for a friend whose position is old (as the map marks it). */
  stale?: boolean;
  style?: StyleProp<ViewStyle>;
}) {
  const theme = useTheme();
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [avatarId]);
  const circle = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View
      style={[
        styles.root,
        circle,
        { backgroundColor: stale ? theme.staleMarker : theme.friendMarker, opacity: stale ? 0.7 : 1 },
        style,
      ]}
    >
      {avatarId && !failed ? (
        <Image source={{ uri: avatarUrl(avatarId) }} style={circle} onError={() => setFailed(true)} />
      ) : (
        <Text style={[styles.initials, { color: theme.friendMarkerText, fontSize: Math.round(size * 0.4) }]}>
          {initialsFor(name)}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: 'center', justifyContent: 'center', overflow: 'hidden' },
  initials: { fontWeight: '700' },
});

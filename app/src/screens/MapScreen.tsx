import { Camera, GeoJSONSource, Layer, Map, UserLocation } from '@maplibre/maplibre-react-native';
import type { CameraRef, PressEventWithFeatures } from '@maplibre/maplibre-react-native';
import { useNavigation } from '@react-navigation/native';
import * as Location from 'expo-location';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { NativeSyntheticEvent } from 'react-native';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Icon } from '../components/Icon';
import { useFriends } from '../friends/useFriends';
import { useLocationStream } from '../location/useLocationStream';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale, timeAgo } from '../time';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };

function initialsFor(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  const first = parts[0][0] ?? '';
  const last = parts.length > 1 ? parts[parts.length - 1][0] : '';
  return (first + last).toUpperCase();
}

// The home screen: a full-screen map of you and your friends. Two buttons —
// Friends (with a badge for requests waiting on you) and "fit everyone".
// Tapping a friend shows who it is and when they were last seen.
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<any>();
  const cameraRef = useRef<CameraRef>(null);
  const streamed = useLocationStream();
  const { friends } = useFriends();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());
  const fittedOnce = useRef(false);

  // Re-render every minute so "last seen" and faded markers stay current.
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60 * 1000);
    return () => clearInterval(t);
  }, []);

  // The stream only ever adds people, so filter by the friends list to drop
  // anyone just removed.
  const acceptedIds = useMemo(
    () => (friends ? new Set(friends.filter((f) => f.status === 'accepted').map((f) => f.friendId)) : null),
    [friends]
  );
  const locations = acceptedIds ? streamed.filter((l) => acceptedIds.has(l.userId)) : streamed;
  const pendingCount = friends?.filter((f) => f.isIncomingRequest).length ?? 0;
  const selected = locations.find((l) => l.userId === selectedId) ?? null;

  const fitEveryone = async (animate = true) => {
    const me = await Location.getLastKnownPositionAsync().catch(() => null);
    const points: [number, number][] = locations.map((l) => [l.longitude, l.latitude]);
    if (me) points.push([me.coords.longitude, me.coords.latitude]);
    if (points.length === 0) return;
    if (points.length === 1) {
      cameraRef.current?.flyTo({ center: points[0], zoom: 14, duration: animate ? 800 : 0 });
      return;
    }
    const lngs = points.map((p) => p[0]);
    const lats = points.map((p) => p[1]);
    cameraRef.current?.fitBounds([Math.min(...lngs), Math.min(...lats), Math.max(...lngs), Math.max(...lats)], {
      padding: FIT_PADDING,
      duration: animate ? 800 : 0,
    });
  };

  // Start on your own position, then frame everyone once their positions
  // first arrive.
  useEffect(() => {
    (async () => {
      const me = await Location.getLastKnownPositionAsync().catch(() => null);
      if (me && !fittedOnce.current) {
        cameraRef.current?.jumpTo({ center: [me.coords.longitude, me.coords.latitude], zoom: 13 });
      }
    })();
  }, []);
  useEffect(() => {
    if (fittedOnce.current || locations.length === 0) return;
    fittedOnce.current = true;
    fitEveryone(false);
  }, [locations.length]); // eslint-disable-line react-hooks/exhaustive-deps

  const onFriendPress = (e: NativeSyntheticEvent<PressEventWithFeatures>) => {
    const userId = e.nativeEvent.features?.[0]?.properties?.userId;
    const friend = locations.find((l) => l.userId === userId);
    if (!friend) return;
    setSelectedId(friend.userId);
    cameraRef.current?.flyTo({ center: [friend.longitude, friend.latitude], zoom: 15, duration: 800 });
  };

  const onMapPress = (e: NativeSyntheticEvent<any>) => {
    if (e.nativeEvent?.features?.length) return; // a friend tap bubbling up
    setSelectedId(null);
  };

  return (
    <View style={styles.root}>
      <Map
        style={styles.map}
        mapStyle={theme.mapStyleUrl}
        compass={false}
        logo={false}
        attributionPosition={{ bottom: 8, left: 8 }}
        onPress={onMapPress}
      >
        <Camera ref={cameraRef} />
        <UserLocation accuracy heading />
        <GeoJSONSource
          id="friends"
          data={{
            type: 'FeatureCollection',
            features: locations.map((l) => ({
              type: 'Feature',
              id: l.userId,
              geometry: { type: 'Point', coordinates: [l.longitude, l.latitude] },
              properties: {
                userId: l.userId,
                initials: initialsFor(l.displayName),
                stale: isStale(l.recordedAt, now),
              },
            })),
          }}
          onPress={onFriendPress}
        >
          <Layer
            id="friend-points"
            type="circle"
            paint={{
              'circle-radius': 15,
              'circle-color': ['case', ['get', 'stale'], theme.staleMarker, theme.friendMarker],
              'circle-stroke-width': 2,
              'circle-stroke-color': theme.markerStroke,
              'circle-opacity': ['case', ['get', 'stale'], 0.7, 1],
            }}
          />
          <Layer
            id="friend-initials"
            type="symbol"
            layout={{
              'text-field': ['get', 'initials'],
              'text-font': ['Noto Sans Bold'],
              'text-size': 12,
              'text-allow-overlap': true,
              'text-ignore-placement': true,
            }}
            paint={{ 'text-color': theme.friendMarkerText }}
          />
        </GeoJSONSource>
      </Map>

      <View style={[styles.topRight, { top: insets.top + 12 }]}>
        <Pressable
          style={styles.fab}
          onPress={() => navigation.navigate('Friends')}
          accessibilityLabel={pendingCount ? `Friends, ${pendingCount} requests waiting` : 'Friends'}
        >
          <Icon name="friends" color={theme.text} size={22} />
          {pendingCount > 0 && (
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{pendingCount}</Text>
            </View>
          )}
        </Pressable>
        <Pressable style={[styles.fab, styles.fabGap]} onPress={() => fitEveryone()} accessibilityLabel="Show everyone">
          <Icon name="fit" color={theme.text} size={22} />
        </Pressable>
      </View>

      {selected && (
        <View style={[styles.card, { bottom: insets.bottom + 24 }]}>
          <Text style={styles.cardName}>{selected.displayName || 'Friend'}</Text>
          <Text style={styles.cardMeta}>Seen {timeAgo(selected.recordedAt, now)}</Text>
        </View>
      )}

      {friends && friends.length === 0 && !selected && (
        <Pressable style={[styles.card, { bottom: insets.bottom + 24 }]} onPress={() => navigation.navigate('Friends')}>
          <Text style={styles.cardName}>No friends yet</Text>
          <Text style={styles.cardMeta}>Tap to invite someone. You'll see each other here.</Text>
        </Pressable>
      )}
    </View>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    map: { flex: 1 },
    topRight: { position: 'absolute', right: 16 },
    fab: {
      width: 48,
      height: 48,
      borderRadius: 24,
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 3 },
      shadowOpacity: 0.25,
      shadowRadius: 8,
      elevation: 5,
    },
    fabGap: { marginTop: 12 },
    badge: {
      position: 'absolute',
      top: -4,
      right: -4,
      minWidth: 20,
      height: 20,
      borderRadius: 10,
      paddingHorizontal: 5,
      backgroundColor: theme.danger,
      alignItems: 'center',
      justifyContent: 'center',
    },
    badgeText: { color: '#fff', fontSize: 12, fontWeight: '700' },
    card: {
      position: 'absolute',
      left: 16,
      right: 16,
      backgroundColor: theme.surface,
      borderColor: theme.border,
      borderWidth: 1,
      borderRadius: 16,
      padding: 16,
    },
    cardName: { color: theme.text, fontSize: 17, fontWeight: '600' },
    cardMeta: { color: theme.textMuted, marginTop: 4 },
  });

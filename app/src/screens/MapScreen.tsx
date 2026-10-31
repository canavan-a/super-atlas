import { Camera, Map, Marker } from '@maplibre/maplibre-react-native';
import type { CameraRef, ViewStateChangeEvent } from '@maplibre/maplibre-react-native';
import { useEffect, useRef, useState, type ReactElement } from 'react';
import type { NativeSyntheticEvent } from 'react-native';
import { StyleSheet, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { Avatar } from '../components/Avatar';
import { MapChrome } from '../components/MapChrome';
import { GLIDE_MS, shouldGlide, useGlide, type LngLat } from '../map/glide';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { useMapData } from './useMapData';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };
// Street level: how close the map opens on you.
const CLOSE_ZOOM = 16;
// "Fit everyone" also turns the map back north-up and flat.
const NEUTRAL = { bearing: 0, pitch: 0 };
// Who the camera is following: you, a friend's userId, or no one.
type Follow = 'me' | string | null;

// The home screen: a full-screen map of you and your friends. Two buttons —
// Friends (with a badge for requests waiting on you) and "fit everyone".
// Tapping someone (you included) follows them at the current zoom until you
// move the map yourself; tapping a friend also shows who it is and when they
// were last seen. Dots slide to new positions rather than jumping.
// (MapScreen.web.tsx is the browser's version.)
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { user } = useAuth();
  const cameraRef = useRef<CameraRef>(null);
  const { locations, me, now, pendingCount, noFriends, selected, setSelectedId } = useMapData();
  const centeredOnce = useRef(false);
  const [mapReady, setMapReady] = useState(false);
  const [follow, setFollow] = useState<Follow>(null);

  const fitEveryone = (animate = true) => {
    setFollow(null);
    const points: [number, number][] = locations.map((l) => [l.longitude, l.latitude]);
    if (me) points.push([me.longitude, me.latitude]);
    if (points.length === 0) return;
    if (points.length === 1) {
      cameraRef.current?.flyTo({ center: points[0], zoom: CLOSE_ZOOM, ...NEUTRAL, duration: animate ? 800 : 0 });
      return;
    }
    const lngs = points.map((p) => p[0]);
    const lats = points.map((p) => p[1]);
    cameraRef.current?.fitBounds([Math.min(...lngs), Math.min(...lats), Math.max(...lngs), Math.max(...lats)], {
      padding: FIT_PADDING,
      ...NEUTRAL,
      duration: animate ? 800 : 0,
    });
  };

  // Open close in on your own position, as soon as there is one and the map
  // has loaded (a camera move sent before that is dropped, leaving the map
  // zoomed all the way out). "Fit everyone" zooms out to friends.
  useEffect(() => {
    if (centeredOnce.current || !me || !mapReady) return;
    centeredOnce.current = true;
    cameraRef.current?.jumpTo({ center: [me.longitude, me.latitude], zoom: CLOSE_ZOOM });
  }, [me, mapReady]);

  // Keep the followed person centered, without touching the zoom: ease over
  // to them when following starts, then slide along with their dot (same
  // time and pace as useGlide) each time they move.
  const followed = follow === 'me' ? me : locations.find((l) => l.userId === follow);
  const followLng = followed?.longitude;
  const followLat = followed?.latitude;
  const lastFollow = useRef<{ who: Follow; at: LngLat } | null>(null);
  useEffect(() => {
    if (follow === null || followLng === undefined || followLat === undefined) {
      lastFollow.current = null;
      return;
    }
    const at: LngLat = [followLng, followLat];
    const prev = lastFollow.current;
    lastFollow.current = { who: follow, at };
    if (prev?.who !== follow) {
      cameraRef.current?.easeTo({ center: at, duration: 600 });
    } else if (shouldGlide(prev.at, at)) {
      cameraRef.current?.easeTo({ center: at, duration: GLIDE_MS, easing: 'linear' });
    } else {
      cameraRef.current?.jumpTo({ center: at });
    }
  }, [follow, followLng, followLat]);

  const onFriendPress = (userId: string) => {
    setSelectedId(userId);
    setFollow(userId);
  };

  const onMePress = () => {
    setSelectedId(null);
    setFollow('me');
  };

  const onMapPress = (e: NativeSyntheticEvent<any>) => {
    if (e.nativeEvent?.features?.length) return; // a marker tap bubbling up
    setSelectedId(null);
    setFollow(null);
  };

  // Dragging, pinching or rotating the map yourself stops following.
  const onRegionWillChange = (e: NativeSyntheticEvent<ViewStateChangeEvent>) => {
    if (e.nativeEvent.userInteraction) setFollow(null);
  };

  return (
    <View style={styles.root}>
      <Map
        style={styles.map}
        mapStyle={theme.mapStyle}
        compass={false}
        logo={false}
        attributionPosition={{ bottom: 8, left: 8 }}
        onPress={onMapPress}
        onRegionWillChange={onRegionWillChange}
        onDidFinishLoadingMap={() => setMapReady(true)}
      >
        <Camera ref={cameraRef} />
        {/* You: your profile picture (or initials) ringed in the theme color
            with a soft halo, drawn first so friends' markers sit on top. */}
        {me && (
          <GlidingMarker id="me" lngLat={[me.longitude, me.latitude]} onPress={onMePress}>
            <View style={styles.meHalo}>
              <Avatar
                name={user?.displayName ?? ''}
                avatarId={user?.avatarId}
                size={30}
                style={[styles.marker, { borderColor: theme.primary }]}
              />
            </View>
          </GlidingMarker>
        )}
        {/* Friends: their profile picture, or their initials, in a ringed
            circle (Avatar), faded once their position is old. */}
        {locations.map((l) => (
          <GlidingMarker
            key={l.userId}
            id={`friend-${l.userId}`}
            lngLat={[l.longitude, l.latitude]}
            onPress={() => onFriendPress(l.userId)}
          >
            <Avatar
              name={l.displayName}
              avatarId={l.avatarId}
              size={30}
              stale={isStale(l.recordedAt, now)}
              style={[styles.marker, { borderColor: theme.markerStroke }]}
            />
          </GlidingMarker>
        ))}
      </Map>

      <MapChrome
        pendingCount={pendingCount}
        onFit={() => fitEveryone()}
        selected={selected}
        now={now}
        noFriends={noFriends}
      />
    </View>
  );
}

// A map marker that slides to each new position instead of jumping (glide.ts).
function GlidingMarker({
  id,
  lngLat,
  onPress,
  children,
}: {
  id: string;
  lngLat: LngLat;
  onPress: () => void;
  children: ReactElement;
}) {
  const shown = useGlide(lngLat);
  return (
    <Marker id={id} lngLat={shown} onPress={onPress}>
      {children}
    </Marker>
  );
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    map: { flex: 1 },
    marker: { borderWidth: 2 },
    meHalo: {
      width: 44,
      height: 44,
      borderRadius: 22,
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: theme.primary + '33',
    },
  });

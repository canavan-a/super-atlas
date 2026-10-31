import { Camera, Map, Marker } from '@maplibre/maplibre-react-native';
import type { CameraRef, LngLatBounds, ViewStateChangeEvent } from '@maplibre/maplibre-react-native';
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
// Town level: tapping someone zooms in to at least this, never out.
const TOWN_ZOOM = 13;
// "Fit everyone" also turns the map back north-up and flat.
const NEUTRAL = { bearing: 0, pitch: 0 };
// Who the camera is following: you, a friend's userId, or no one.
type Follow = 'me' | string | null;

// The home screen: a full-screen map of you and your friends. Two buttons —
// Friends (with a badge for requests waiting on you) and "fit everyone",
// which turns into "show me" once it has zoomed out, to toggle back.
// Tapping someone (you included) turns the map north-up and follows them,
// zooming in to town level if the map is further out than that (and
// otherwise keeping the zoom), until you move the map yourself, and shows a
// card at the bottom: "You", or the friend's picture and when they were last
// seen. Nudging the map keeps the card; moving them off screen closes it.
// "Show on map" in the friends list opens the map following that friend. Dots
// slide to new positions rather than jumping.
// (MapScreen.web.tsx is the browser's version.)
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { user } = useAuth();
  const cameraRef = useRef<CameraRef>(null);
  const { locations, me, now, pendingCount, noFriends, selected, meSelected, setSelectedId, focus, focusDone } =
    useMapData();
  const centeredOnce = useRef(false);
  const [mapReady, setMapReady] = useState(false);
  const [follow, setFollow] = useState<Follow>(null);
  // The map's zoom as of its last move.
  const zoom = useRef(0);
  // Just zoomed out to everyone: the fit button now says "Show me" instead.
  const [fitted, setFitted] = useState(false);

  const fitEveryone = (animate = true) => {
    setFollow(null);
    const points: [number, number][] = locations.map((l) => [l.longitude, l.latitude]);
    if (me) points.push([me.longitude, me.latitude]);
    if (points.length === 0) return;
    if (points.length === 1) {
      cameraRef.current?.flyTo({ center: points[0], zoom: CLOSE_ZOOM, ...NEUTRAL, duration: animate ? 800 : 0 });
      return;
    }
    if (me) setFitted(true);
    const lngs = points.map((p) => p[0]);
    const lats = points.map((p) => p[1]);
    cameraRef.current?.fitBounds([Math.min(...lngs), Math.min(...lats), Math.max(...lngs), Math.max(...lats)], {
      padding: FIT_PADDING,
      ...NEUTRAL,
      duration: animate ? 800 : 0,
    });
  };

  // "Show me", the fit button's other half: back in on you at town level, as
  // close as tapping someone goes.
  const zoomToMe = () => {
    setFitted(false);
    setFollow(null);
    if (!me) return;
    cameraRef.current?.flyTo({ center: [me.longitude, me.latitude], zoom: TOWN_ZOOM, ...NEUTRAL, duration: 800 });
  };

  // Open close in on your own position, as soon as there is one and the map
  // has loaded (a camera move sent before that is dropped, leaving the map
  // zoomed all the way out). "Fit everyone" zooms out to friends.
  useEffect(() => {
    if (centeredOnce.current || !me || !mapReady) return;
    centeredOnce.current = true;
    cameraRef.current?.jumpTo({ center: [me.longitude, me.latitude], zoom: CLOSE_ZOOM });
  }, [me, mapReady]);

  // Keep the followed person centered: ease over to them when following
  // starts (north-up, and zoomed in to town level if further out), then slide
  // along with their dot (same time and pace as useGlide) each time they move.
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
      cameraRef.current?.easeTo({
        center: at,
        ...(zoom.current < TOWN_ZOOM && { zoom: TOWN_ZOOM }),
        bearing: 0,
        duration: 600,
      });
    } else if (shouldGlide(prev.at, at)) {
      cameraRef.current?.easeTo({ center: at, duration: GLIDE_MS, easing: 'linear' });
    } else {
      cameraRef.current?.jumpTo({ center: at });
    }
  }, [follow, followLng, followLat]);

  // Opened from "Show on map": follow that friend, as if they'd been tapped.
  const focusUserId = focus?.userId;
  useEffect(() => {
    if (!focusUserId || !mapReady) return;
    setSelectedId(focusUserId);
    setFollow(focusUserId);
    setFitted(false);
    focusDone();
  }, [focusUserId, mapReady]); // eslint-disable-line react-hooks/exhaustive-deps

  const onFriendPress = (userId: string) => {
    setFitted(false);
    setSelectedId(userId);
    setFollow(userId);
  };

  const onMePress = () => {
    setFitted(false);
    setSelectedId('me');
    setFollow('me');
  };

  const onMapPress = (e: NativeSyntheticEvent<any>) => {
    // A marker's press bubbles up to the map as an onPress too, carrying the
    // marker's id; without this it undid the follow the marker just started.
    if (e.nativeEvent?.id !== undefined || e.nativeEvent?.features?.length) return;
    setSelectedId(null);
    setFollow(null);
  };

  // Dragging, pinching or rotating the map yourself stops following.
  const movedByUser = useRef(false);
  const onRegionWillChange = (e: NativeSyntheticEvent<ViewStateChangeEvent>) => {
    if (!e.nativeEvent.userInteraction) return;
    movedByUser.current = true;
    setFollow(null);
  };

  // A nudge keeps the card up; moving the map so far that whoever it's for
  // is off screen closes it.
  const shownPerson = meSelected ? me : selected;
  const onRegionDidChange = (e: NativeSyntheticEvent<ViewStateChangeEvent>) => {
    zoom.current = e.nativeEvent.zoom;
    if (!movedByUser.current) return;
    movedByUser.current = false;
    if (shownPerson && !inBounds(e.nativeEvent.bounds, shownPerson.longitude, shownPerson.latitude)) {
      setSelectedId(null);
    }
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
        onRegionDidChange={onRegionDidChange}
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
        onLocate={fitted ? zoomToMe : undefined}
        selected={selected}
        meSelected={meSelected}
        me={me}
        now={now}
        noFriends={noFriends}
      />
    </View>
  );
}

// Whether a point is inside the map's visible [west, south, east, north],
// including a view that spans the antimeridian (west > east).
function inBounds([west, south, east, north]: LngLatBounds, lng: number, lat: number) {
  if (lat < south || lat > north) return false;
  return west <= east ? lng >= west && lng <= east : lng >= west || lng <= east;
}

// A map marker that slides to each new position instead of jumping (glide.ts),
// with an invisible margin around it so a slightly-off tap still lands.
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
      <View style={hitArea}>{children}</View>
    </Marker>
  );
}

// How far past a marker's edge a tap still counts as on it.
const hitArea = { padding: 10 };

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

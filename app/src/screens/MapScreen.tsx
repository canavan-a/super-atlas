import { Camera, GeoJSONSource, Layer, Map, Marker } from '@maplibre/maplibre-react-native';
import type { CameraRef } from '@maplibre/maplibre-react-native';
import { useEffect, useRef, useState } from 'react';
import type { NativeSyntheticEvent } from 'react-native';
import { StyleSheet, View } from 'react-native';
import { Avatar } from '../components/Avatar';
import { MapChrome } from '../components/MapChrome';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { useMapData } from './useMapData';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };
// Street level: how close the map opens on you, and flies to a friend.
const CLOSE_ZOOM = 16;
// "Fit everyone" also turns the map back north-up and flat.
const NEUTRAL = { bearing: 0, pitch: 0 };

// The home screen: a full-screen map of you and your friends. Two buttons —
// Friends (with a badge for requests waiting on you) and "fit everyone".
// Tapping a friend shows who it is and when they were last seen.
// (MapScreen.web.tsx is the browser's version.)
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const cameraRef = useRef<CameraRef>(null);
  const { locations, me, now, pendingCount, noFriends, selected, setSelectedId } = useMapData();
  const centeredOnce = useRef(false);
  const [mapReady, setMapReady] = useState(false);

  const fitEveryone = (animate = true) => {
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

  const onFriendPress = (userId: string) => {
    const friend = locations.find((l) => l.userId === userId);
    if (!friend) return;
    setSelectedId(friend.userId);
    cameraRef.current?.flyTo({ center: [friend.longitude, friend.latitude], zoom: CLOSE_ZOOM, duration: 800 });
  };

  const onMapPress = (e: NativeSyntheticEvent<any>) => {
    if (e.nativeEvent?.features?.length) return; // a friend tap bubbling up
    setSelectedId(null);
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
        onDidFinishLoadingMap={() => setMapReady(true)}
      >
        <Camera ref={cameraRef} />
        {/* You: a small dot with a soft halo, under friends' larger initialed
            markers so they stay tappable. */}
        <GeoJSONSource
          id="me"
          data={{
            type: 'FeatureCollection',
            features: me
              ? [{ type: 'Feature', properties: {}, geometry: { type: 'Point', coordinates: [me.longitude, me.latitude] } }]
              : [],
          }}
        >
          <Layer
            id="me-halo"
            type="circle"
            paint={{ 'circle-radius': 22, 'circle-color': theme.primary, 'circle-opacity': 0.2 }}
          />
          <Layer
            id="me-dot"
            type="circle"
            paint={{
              'circle-radius': 8,
              'circle-color': theme.primary,
              'circle-stroke-width': 3,
              'circle-stroke-color': theme.markerStroke,
            }}
          />
        </GeoJSONSource>
        {/* Friends: their profile picture, or their initials, in a ringed
            circle (Avatar), faded once their position is old. */}
        {locations.map((l) => (
          <Marker key={l.userId} id={`friend-${l.userId}`} lngLat={[l.longitude, l.latitude]} onPress={() => onFriendPress(l.userId)}>
            <Avatar
              name={l.displayName}
              avatarId={l.avatarId}
              size={30}
              stale={isStale(l.recordedAt, now)}
              style={[styles.marker, { borderColor: theme.markerStroke }]}
            />
          </Marker>
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

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    map: { flex: 1 },
    marker: { borderWidth: 2 },
  });

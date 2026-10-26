import { Camera, GeoJSONSource, Layer, Map, UserLocation } from '@maplibre/maplibre-react-native';
import type { CameraRef, PressEventWithFeatures } from '@maplibre/maplibre-react-native';
import * as Location from 'expo-location';
import { useEffect, useRef } from 'react';
import type { NativeSyntheticEvent } from 'react-native';
import { StyleSheet, View } from 'react-native';
import { MapChrome, initialsFor } from '../components/MapChrome';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { useMapData } from './useMapData';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };

// The home screen: a full-screen map of you and your friends. Two buttons —
// Friends (with a badge for requests waiting on you) and "fit everyone".
// Tapping a friend shows who it is and when they were last seen.
// (MapScreen.web.tsx is the browser's version.)
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const cameraRef = useRef<CameraRef>(null);
  const { locations, now, pendingCount, noFriends, selected, setSelectedId } = useMapData();
  const fittedOnce = useRef(false);

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
        mapStyle={theme.mapStyle}
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
  });

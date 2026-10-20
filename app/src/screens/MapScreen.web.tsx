import maplibregl from 'maplibre-gl';
import { useEffect, useRef } from 'react';
import { StyleSheet, View } from 'react-native';
import { MapChrome, initialsFor } from '../components/MapChrome';
import type { FriendLocation } from '../location/friendStream';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { useMapData } from './useMapData';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };

// The browser's map, on maplibre-gl with the same basemaps, markers and
// buttons as the Android map (MapScreen.tsx). It only ever shows friends:
// the web build never asks for or sends this device's location, and says so.
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { locations, now, pendingCount, noFriends, selected, setSelectedId } = useMapData();
  const container = useRef<View>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const markers = useRef(new Map<string, maplibregl.Marker>());
  const fittedOnce = useRef(false);
  const latest = useRef<FriendLocation[]>(locations);
  latest.current = locations;

  const fitEveryone = (animate = true) => {
    const m = map.current;
    const points = latest.current;
    if (!m || points.length === 0) return;
    if (points.length === 1) {
      m.flyTo({ center: [points[0].longitude, points[0].latitude], zoom: 14, animate });
      return;
    }
    const bounds = new maplibregl.LngLatBounds();
    for (const p of points) bounds.extend([p.longitude, p.latitude]);
    m.fitBounds(bounds, { padding: FIT_PADDING, animate });
  };

  // Create the map once; tapping empty map closes the friend card.
  useEffect(() => {
    const el = container.current as unknown as HTMLElement | null;
    if (!el) return;
    const m = new maplibregl.Map({
      container: el,
      style: theme.mapStyleUrl,
      center: [0, 20],
      zoom: 1.5,
      attributionControl: { compact: true },
    });
    m.on('click', () => setSelectedId(null));
    map.current = m;
    const owned = markers.current;
    return () => {
      owned.forEach((marker) => marker.remove());
      owned.clear();
      m.remove();
      map.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    map.current?.setStyle(theme.mapStyleUrl);
  }, [theme.mapStyleUrl]);

  // Keep one marker per friend, in step with their positions.
  useEffect(() => {
    const m = map.current;
    if (!m) return;
    const seen = new Set<string>();
    for (const l of locations) {
      seen.add(l.userId);
      let marker = markers.current.get(l.userId);
      if (!marker) {
        const el = document.createElement('button');
        el.type = 'button';
        el.addEventListener('click', (e) => {
          e.stopPropagation();
          const friend = latest.current.find((f) => f.userId === l.userId);
          if (!friend) return;
          setSelectedId(friend.userId);
          m.flyTo({ center: [friend.longitude, friend.latitude], zoom: 15 });
        });
        marker = new maplibregl.Marker({ element: el }).setLngLat([l.longitude, l.latitude]).addTo(m);
        markers.current.set(l.userId, marker);
      }
      marker.setLngLat([l.longitude, l.latitude]);
      styleMarker(marker.getElement(), l, theme, isStale(l.recordedAt, now));
    }
    for (const [id, marker] of markers.current) {
      if (!seen.has(id)) {
        marker.remove();
        markers.current.delete(id);
      }
    }
    if (!fittedOnce.current && locations.length > 0) {
      fittedOnce.current = true;
      fitEveryone(false);
    }
  }, [locations, theme, now]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <View style={styles.root}>
      <View ref={container} style={styles.map} />
      <MapChrome
        pendingCount={pendingCount}
        onFit={() => fitEveryone()}
        selected={selected}
        now={now}
        noFriends={noFriends}
        notice="View only — the web version doesn't share your location."
      />
    </View>
  );
}

// The same marker as the Android map: a circle with the friend's initials,
// faded once their position is over an hour old.
function styleMarker(el: HTMLElement, l: FriendLocation, theme: Theme, stale: boolean) {
  el.textContent = initialsFor(l.displayName);
  el.title = l.displayName;
  el.setAttribute('aria-label', l.displayName || 'Friend');
  Object.assign(el.style, {
    width: '30px',
    height: '30px',
    borderRadius: '15px',
    border: `2px solid ${theme.markerStroke}`,
    background: stale ? theme.staleMarker : theme.friendMarker,
    opacity: stale ? '0.7' : '1',
    color: theme.friendMarkerText,
    font: '700 12px system-ui, sans-serif',
    cursor: 'pointer',
    padding: '0',
  });
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    map: { flex: 1 },
  });

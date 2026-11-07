import maplibregl from 'maplibre-gl';
import { useEffect, useRef, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { avatarUrl, initialsFor } from '../components/Avatar';
import { MapChrome, type GroupPerson } from '../components/MapChrome';
import type { FriendLocation } from '../location/friendStream';
import { inMiddle } from '../map/bounds';
import { clusterPoints, MAX_SPLIT_ZOOM, type Cluster } from '../map/cluster';
import { GLIDE_MS, glide, shouldGlide, type LngLat } from '../map/glide';
import { Theme, useTheme, useThemedStyles } from '../theme';
import { isStale } from '../time';
import { useMapData } from './useMapData';

const FIT_PADDING = { top: 140, right: 80, bottom: 160, left: 80 };
// "Fit everyone" also turns the map back north-up and flat.
const NEUTRAL = { bearing: 0, pitch: 0 };
// Town level: tapping someone zooms in to at least this, and turns the map
// north-up (see MapScreen.tsx).
const TOWN_ZOOM = 13;
const START_ZOOM = 1.5;

// The browser's map, on maplibre-gl with the same basemaps, markers and
// buttons as the Android map (MapScreen.tsx), groups of nearby friends
// included. It only ever shows friends: the web build never asks for or
// sends this device's location, and says so.
export default function MapScreen() {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const { locations, now, pendingCount, noFriends, selected, setSelectedId, focus, focusDone } = useMapData();
  const container = useRef<View>(null);
  const map = useRef<maplibregl.Map | null>(null);
  // The friend whose card is up, for the map's own handlers.
  const shown = useRef(selected);
  shown.current = selected;
  // One marker per friend standing alone, or per group (by cluster key).
  const markers = useRef(new Map<string, maplibregl.Marker>());
  const fittedOnce = useRef(false);
  const latest = useRef<FriendLocation[]>(locations);
  latest.current = locations;
  // Each marker's slide in progress (glide.ts), stopped before the next.
  const glides = useRef(new Map<string, () => void>());
  // Each marker's latest position (where its slide ends).
  const targets = useRef(new Map<string, LngLat>());
  // The friend the camera follows (see MapScreen.tsx), or null, and where
  // they were last.
  const following = useRef<string | null>(null);
  const followAt = useRef<LngLat | null>(null);
  // The map's zoom as of its last move, for grouping.
  const [zoom, setZoom] = useState(START_ZOOM);
  // The groups on the map now, by key, for their markers' taps.
  const clusters = useRef(new Map<string, Cluster<FriendLocation>>());
  // A tapped group too close together to zoom apart: who's in it.
  const [groupIds, setGroupIds] = useState<string[] | null>(null);
  const group = groupIds ? locations.filter((l) => groupIds.includes(l.userId)) : null;
  const groupRef = useRef(group);
  groupRef.current = group;

  const fitEveryone = (animate = true) => {
    following.current = null;
    setGroupIds(null);
    const m = map.current;
    const points = latest.current;
    if (!m || points.length === 0) return;
    if (points.length === 1) {
      m.flyTo({ center: [points[0].longitude, points[0].latitude], zoom: 16, ...NEUTRAL, animate });
      return;
    }
    const bounds = new maplibregl.LngLatBounds();
    for (const p of points) bounds.extend([p.longitude, p.latitude]);
    m.fitBounds(bounds, { padding: FIT_PADDING, ...NEUTRAL, animate });
  };

  const followFriend = (friend: FriendLocation) => {
    const m = map.current;
    if (!m) return;
    setGroupIds(null);
    setSelectedId(friend.userId);
    following.current = friend.userId;
    followAt.current = [friend.longitude, friend.latitude];
    m.easeTo({
      center: [friend.longitude, friend.latitude],
      zoom: Math.max(m.getZoom(), TOWN_ZOOM),
      bearing: 0,
      duration: 600,
    });
  };

  // A group: zoom in just far enough that everyone in it stands alone, or,
  // when that's closer than street level, list them instead (as Android).
  const openGroup = (c: Cluster<FriendLocation>) => {
    const m = map.current;
    if (!m) return;
    following.current = null;
    setSelectedId(null);
    if (c.splitZoom <= MAX_SPLIT_ZOOM) {
      setGroupIds(null);
      m.easeTo({ center: c.lngLat, zoom: c.splitZoom + 0.3, duration: 600 });
    } else {
      setGroupIds(c.members.map((l) => l.userId));
      m.easeTo({ center: c.lngLat, duration: 600 });
    }
  };

  // Create the map once; tapping empty map closes the friend or group card,
  // and it or moving the map yourself stops following. A move that leaves
  // whoever the card is for out of the middle of the screen (bounds.ts)
  // closes it too (a nudge doesn't).
  useEffect(() => {
    const el = container.current as unknown as HTMLElement | null;
    if (!el) return;
    const m = new maplibregl.Map({
      container: el,
      style: theme.mapStyle,
      center: [0, 20],
      zoom: START_ZOOM,
      attributionControl: { compact: true },
    });
    m.on('click', () => {
      setSelectedId(null);
      setGroupIds(null);
      following.current = null;
    });
    m.on('movestart', (e) => {
      if (e.originalEvent) following.current = null;
    });
    m.on('moveend', (e) => {
      setZoom(m.getZoom());
      if (!e.originalEvent) return;
      const b = m.getBounds();
      const box: [number, number, number, number] = [b.getWest(), b.getSouth(), b.getEast(), b.getNorth()];
      const s = shown.current;
      if (s && !inMiddle(box, s.longitude, s.latitude)) setSelectedId(null);
      const g = groupRef.current?.[0];
      if (g && !inMiddle(box, g.longitude, g.latitude)) setGroupIds(null);
    });
    map.current = m;
    const owned = markers.current;
    const sliding = glides.current;
    return () => {
      sliding.forEach((stop) => stop());
      sliding.clear();
      owned.forEach((marker) => marker.remove());
      owned.clear();
      m.remove();
      map.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    map.current?.setStyle(theme.mapStyle);
  }, [theme.mapStyle]);

  // Opened from "Show on map": follow that friend, as if they'd been tapped.
  useEffect(() => {
    if (!focus || !map.current) return;
    fittedOnce.current = true; // don't zoom out to everyone over it
    followFriend(focus);
    focusDone();
  }, [focus?.userId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Keep one marker per friend, or per group of friends whose markers would
  // overlap at this zoom (cluster.ts), each sliding to its new position, with
  // the camera sliding along on whoever is followed.
  useEffect(() => {
    const m = map.current;
    if (!m) return;
    const grouped = clusterPoints(
      locations,
      (l) => [l.longitude, l.latitude],
      (l) => l.userId,
      zoom
    );
    clusters.current = new Map(grouped.map((c) => [c.key, c]));
    const seen = new Set<string>();
    for (const c of grouped) {
      seen.add(c.key);
      let marker = markers.current.get(c.key);
      if (!marker) {
        // The visible marker sits inside an invisible margin that also takes
        // the tap, so a slightly-off one still lands.
        const key = c.key;
        const el = document.createElement('div');
        el.style.padding = '10px';
        el.style.cursor = 'pointer';
        const inner = document.createElement('button');
        inner.type = 'button';
        el.appendChild(inner);
        el.addEventListener('click', (e) => {
          e.stopPropagation();
          const current = clusters.current.get(key);
          if (!current) return;
          if (current.members.length > 1) {
            openGroup(current);
            return;
          }
          const friend = latest.current.find((f) => f.userId === current.members[0].userId);
          if (friend) followFriend(friend);
        });
        marker = new maplibregl.Marker({ element: el }).setLngLat(c.lngLat).addTo(m);
        markers.current.set(c.key, marker);
      }
      // Only a new position moves it: this also runs on unrelated re-renders,
      // which mustn't restart a slide in progress.
      const from = marker.getLngLat().toArray() as LngLat;
      const to = c.lngLat;
      const prev = targets.current.get(c.key);
      targets.current.set(c.key, to);
      if (prev && (prev[0] !== to[0] || prev[1] !== to[1])) {
        glides.current.get(c.key)?.();
        if (shouldGlide(from, to)) {
          const target = marker;
          glides.current.set(c.key, glide(from, to, (p) => target.setLngLat(p)));
        } else {
          marker.setLngLat(to);
        }
      }
      const inner = marker.getElement().firstElementChild as HTMLElement;
      if (c.members.length > 1) stylePile(inner, c.members, theme, now);
      else styleFace(inner, c.members[0], theme, isStale(c.members[0].recordedAt, now));
    }
    for (const [id, marker] of markers.current) {
      if (!seen.has(id)) {
        glides.current.get(id)?.();
        glides.current.delete(id);
        targets.current.delete(id);
        marker.remove();
        markers.current.delete(id);
      }
    }

    // The followed friend moved: slide (or jump) the camera along.
    const f = following.current ? locations.find((l) => l.userId === following.current) : undefined;
    if (f) {
      const to: LngLat = [f.longitude, f.latitude];
      const prev = followAt.current;
      followAt.current = to;
      if (prev && (prev[0] !== to[0] || prev[1] !== to[1])) {
        if (shouldGlide(prev, to)) m.easeTo({ center: to, duration: GLIDE_MS, easing: (t) => t });
        else m.jumpTo({ center: to });
      }
    } else {
      followAt.current = null;
    }

    if (!fittedOnce.current && locations.length > 0) {
      fittedOnce.current = true;
      fitEveryone(false);
    }
  }, [locations, theme, now, zoom]); // eslint-disable-line react-hooks/exhaustive-deps

  const groupPeople: GroupPerson[] | null = group
    ? group.map((l) => ({
        id: l.userId,
        name: l.displayName,
        avatarId: l.avatarId,
        stale: isStale(l.recordedAt, now),
        isMe: false,
      }))
    : null;

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
        group={selected ? null : groupPeople}
        onPick={(id) => {
          const friend = latest.current.find((f) => f.userId === id);
          if (friend) followFriend(friend);
        }}
      />
    </View>
  );
}

// The same marker as the Android map (Avatar): a circle with the friend's
// profile picture or initials, faded once their position is over an hour old.
function styleFace(el: HTMLElement, l: FriendLocation, theme: Theme, stale: boolean) {
  el.replaceChildren();
  el.textContent = l.avatarId ? '' : initialsFor(l.displayName);
  el.title = l.displayName;
  el.setAttribute('aria-label', l.displayName || 'Friend');
  Object.assign(el.style, faceStyle(theme, stale, l.avatarId));
  Object.assign(el.style, { cursor: 'pointer', padding: '0', display: 'block', boxShadow: 'none' });
}

// A group, as on Android: up to three faces overlapping on a rounded
// backing, or two and a "+N" past that.
function stylePile(el: HTMLElement, members: FriendLocation[], theme: Theme, now: number) {
  const faces = members.length > 3 ? members.slice(0, 2) : members;
  const children = faces.map((l, i) => {
    const face = document.createElement('div');
    face.textContent = l.avatarId ? '' : initialsFor(l.displayName);
    Object.assign(face.style, faceStyle(theme, isStale(l.recordedAt, now), l.avatarId), {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      marginLeft: i > 0 ? '-12px' : '0',
    });
    return face;
  });
  if (members.length > 3) {
    const more = document.createElement('div');
    more.textContent = `+${members.length - 2}`;
    Object.assign(more.style, faceStyle(theme, false), {
      backgroundColor: theme.primary,
      color: '#fff',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      marginLeft: '-12px',
    });
    children.push(more);
  }
  el.replaceChildren(...children);
  const names = members.map((l) => l.displayName || 'Friend').join(', ');
  el.title = names;
  el.setAttribute('aria-label', `${members.length} friends: ${names}`);
  Object.assign(el.style, {
    width: 'auto',
    height: 'auto',
    display: 'flex',
    padding: '3px',
    borderRadius: '20px',
    border: 'none',
    backgroundColor: theme.surface,
    backgroundImage: 'none',
    opacity: '1',
    boxShadow: '0 2px 6px rgba(0,0,0,0.3)',
    cursor: 'pointer',
  });
}

function faceStyle(theme: Theme, stale: boolean, avatarId?: string): Partial<CSSStyleDeclaration> {
  return {
    width: '30px',
    height: '30px',
    boxSizing: 'border-box',
    borderRadius: '15px',
    border: `2px solid ${theme.markerStroke}`,
    backgroundColor: stale ? theme.staleMarker : theme.friendMarker,
    backgroundImage: avatarId ? `url("${avatarUrl(avatarId)}")` : 'none',
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    opacity: stale ? '0.7' : '1',
    color: theme.friendMarkerText,
    font: '700 12px system-ui, sans-serif',
  };
}

const makeStyles = (theme: Theme) =>
  StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.bg },
    map: { flex: 1 },
  });

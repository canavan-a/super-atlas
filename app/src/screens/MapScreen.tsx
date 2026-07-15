import { Camera, GeoJSONSource, Layer, Map, UserLocation } from '@maplibre/maplibre-react-native';
import type { CameraRef, ViewStateChangeEvent } from '@maplibre/maplibre-react-native';
import type { NativeSyntheticEvent } from 'react-native';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Pressable, Text, View } from 'react-native';
import type { PressEventWithFeatures } from '@maplibre/maplibre-react-native';
import { useFocusEffect } from '@react-navigation/native';
import { useLocationStream } from '../location/useLocationStream';
import { colors } from '../theme/colors';
import { apiJson } from '../api/client';
import * as Location from 'expo-location';

type Friend = { FriendID: string; Status: 'pending' | 'accepted' };

// Free, no-API-key vector style with street-level detail.
// https://openfreemap.org
const MAP_STYLE_URL = 'https://tiles.openfreemap.org/styles/liberty';

const FIT_BOUNDS_PADDING = { top: 120, right: 120, bottom: 120, left: 120 };

function initialsFor(name: string | undefined): string {
  if (!name) return '?';
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] ?? '';
  const last = parts.length > 1 ? parts[parts.length - 1][0] : '';
  return (first + last).toUpperCase();
}

export default function MapScreen() {
  const streamedLocations = useLocationStream();
  const [acceptedFriendIds, setAcceptedFriendIds] = useState<Set<string> | null>(null);
  // Filter out anyone no longer an accepted friend (e.g. just removed) —
  // the WebSocket hook has no concept of unfriending, it only ever merges
  // updates in, so stale last-known positions would otherwise linger forever.
  const friendLocations = acceptedFriendIds
    ? streamedLocations.filter((f) => acceptedFriendIds.has(f.userId))
    : streamedLocations;
  const [namesByUserId, setNamesByUserId] = useState<Record<string, string>>({});
  const [showZoomAll, setShowZoomAll] = useState(true);
  const [followedUserId, setFollowedUserId] = useState<string | null>(null);
  const followed = friendLocations.find((f) => f.userId === followedUserId) ?? null;
  const cameraRef = useRef<CameraRef>(null);
  const isZoomedToMe = useRef(false);

  useFocusEffect(
    useCallback(() => {
      let cancelled = false;
      apiJson<Friend[]>('/friends/')
        .then((list) => {
          if (cancelled) return;
          setAcceptedFriendIds(new Set((list ?? []).filter((f) => f.Status === 'accepted').map((f) => f.FriendID)));
        })
        .catch(() => {});
      return () => {
        cancelled = true;
      };
    }, [])
  );

  useEffect(() => {
    (async () => {
      const position =
        (await Location.getLastKnownPositionAsync()) ?? (await Location.getCurrentPositionAsync().catch(() => null));
      if (!position) return;
      cameraRef.current?.jumpTo({ center: [position.coords.longitude, position.coords.latitude], zoom: 12 });
    })();
  }, []);

  useEffect(() => {
    const missing = friendLocations.map((f) => f.userId).filter((id) => !(id in namesByUserId));
    if (missing.length === 0) return;
    missing.forEach((userId) => {
      setNamesByUserId((prev) => ({ ...prev, [userId]: prev[userId] ?? '' }));
      apiJson<{ userId: string; displayName: string }>(`/friends/name/${userId}`)
        .then((res) => setNamesByUserId((prev) => ({ ...prev, [userId]: res.displayName })))
        .catch(() => {});
    });
  }, [friendLocations, namesByUserId]);

  // Keep the camera centered on the followed friend as their position updates.
  useEffect(() => {
    if (!followed) return;
    cameraRef.current?.easeTo({ center: [followed.longitude, followed.latitude], duration: 800 });
  }, [followed?.longitude, followed?.latitude]);

  const onFriendPress = (e: NativeSyntheticEvent<PressEventWithFeatures>) => {
    const feature = e.nativeEvent.features?.[0];
    const userId = feature?.properties?.userId;
    if (!userId) return;
    const friend = friendLocations.find((f) => f.userId === userId);
    if (!friend) return;

    if (followedUserId === userId) {
      setFollowedUserId(null);
      return;
    }
    setFollowedUserId(userId);
    cameraRef.current?.flyTo({ center: [friend.longitude, friend.latitude], zoom: 15, duration: 800 });
  };

  const onRegionDidChange = (e: NativeSyntheticEvent<ViewStateChangeEvent>) => {
    if (!e.nativeEvent.userInteraction) return;
    // A manual pan/zoom invalidates both "zoom to all" and following a friend.
    setShowZoomAll(true);
    setFollowedUserId(null);
  };

  const onMyLocationPress = async () => {
    const position = await Location.getLastKnownPositionAsync();
    if (!position) return;
    const zoomingIn = !isZoomedToMe.current;
    isZoomedToMe.current = zoomingIn;
    cameraRef.current?.flyTo({
      center: [position.coords.longitude, position.coords.latitude],
      zoom: zoomingIn ? 16 : 12,
      duration: 800,
    });
  };

  const onZoomAllPress = async () => {
    setFollowedUserId(null);
    setShowZoomAll(false);

    if (friendLocations.length === 0) {
      const position = await Location.getLastKnownPositionAsync();
      if (!position) return;
      cameraRef.current?.flyTo({
        center: [position.coords.longitude, position.coords.latitude],
        zoom: 12,
        duration: 800,
      });
      return;
    }

    const lngs = friendLocations.map((f) => f.longitude);
    const lats = friendLocations.map((f) => f.latitude);
    cameraRef.current?.fitBounds(
      [Math.min(...lngs), Math.min(...lats), Math.max(...lngs), Math.max(...lats)],
      { padding: FIT_BOUNDS_PADDING, duration: 800 }
    );
  };

  return (
    <View style={{ flex: 1, backgroundColor: colors.background }}>
      <Map
        style={{ flex: 1 }}
        mapStyle={MAP_STYLE_URL}
        compassPosition={{ top: 8, left: 8 }}
        onRegionDidChange={onRegionDidChange}
      >
        <Camera ref={cameraRef} initialViewState={{ center: [-74.0, 40.7], zoom: 12 }} />

        <UserLocation accuracy heading onPress={onMyLocationPress} />

        <GeoJSONSource
          id="friends"
          data={{
            type: 'FeatureCollection',
            features: friendLocations.map((f) => ({
              type: 'Feature',
              id: f.userId,
              geometry: { type: 'Point', coordinates: [f.longitude, f.latitude] },
              properties: { userId: f.userId, initials: initialsFor(namesByUserId[f.userId]) },
            })),
          }}
          onPress={onFriendPress}
        >
          <Layer
            id="friend-points-glow"
            type="circle"
            paint={{
              'circle-radius': 20,
              'circle-color': colors.accent,
              'circle-opacity': 0.25,
              'circle-blur': 0.9,
            }}
          />
          <Layer
            id="friend-points"
            type="circle"
            paint={{
              'circle-radius': 13,
              'circle-color': colors.accent,
              'circle-stroke-width': 1.5,
              'circle-stroke-color': '#FFFFFF',
              'circle-stroke-opacity': 0.85,
            }}
          />
          <Layer
            id="friend-initials"
            type="symbol"
            layout={{
              'text-field': ['get', 'initials'],
              'text-font': ['Noto Sans Bold'],
              'text-size': 11,
              'text-allow-overlap': true,
              'text-ignore-placement': true,
            }}
            paint={{
              'text-color': '#fff',
            }}
          />
        </GeoJSONSource>
      </Map>

      {showZoomAll && (
        <Pressable
          onPress={onZoomAllPress}
          style={{
            position: 'absolute',
            top: 16,
            right: 16,
            width: 46,
            height: 46,
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: colors.surfaceElevated,
            borderRadius: 23,
            shadowColor: '#000',
            shadowOffset: { width: 0, height: 4 },
            shadowOpacity: 0.35,
            shadowRadius: 10,
            elevation: 6,
          }}
        >
          <Text style={{ color: colors.text, fontSize: 20, lineHeight: 22 }}>⛶</Text>
        </Pressable>
      )}

      {followed && (
        <View
          style={{
            position: 'absolute',
            left: 16,
            right: 16,
            bottom: 24,
            backgroundColor: colors.surfaceElevated,
            borderColor: colors.border,
            borderWidth: 1,
            borderRadius: 16,
            padding: 16,
          }}
        >
          <Text style={{ color: colors.text, fontSize: 16, fontWeight: '600' }}>
            {namesByUserId[followed.userId] || 'Friend'}
          </Text>
          <Text style={{ color: colors.textMuted, marginTop: 4 }}>
            Last seen {new Date(followed.recordedAt).toLocaleTimeString()}
          </Text>
        </View>
      )}
    </View>
  );
}

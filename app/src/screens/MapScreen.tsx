import { Camera, GeoJSONSource, Layer, Map } from '@maplibre/maplibre-react-native';
import type { NativeSyntheticEvent } from 'react-native';
import { useState } from 'react';
import { Text, View } from 'react-native';
import type { PressEventWithFeatures } from '@maplibre/maplibre-react-native';
import { useLocationStream } from '../location/useLocationStream';
import { colors } from '../theme/colors';

// Free, no-API-key vector style. Swap for a self-hosted style URL if you
// want full control over the look.
const MAP_STYLE_URL = 'https://demotiles.maplibre.org/style.json';

export default function MapScreen() {
  const friendLocations = useLocationStream();
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const selected = friendLocations.find((f) => f.userId === selectedUserId) ?? null;

  const onFriendPress = (e: NativeSyntheticEvent<PressEventWithFeatures>) => {
    const feature = e.nativeEvent.features?.[0];
    const userId = feature?.properties?.userId;
    if (userId) setSelectedUserId(userId);
  };

  return (
    <View style={{ flex: 1, backgroundColor: colors.background }}>
      <Map style={{ flex: 1 }} mapStyle={MAP_STYLE_URL}>
        <Camera initialViewState={{ center: [-74.0, 40.7], zoom: 12 }} />

        <GeoJSONSource
          id="friends"
          data={{
            type: 'FeatureCollection',
            features: friendLocations.map((f) => ({
              type: 'Feature',
              id: f.userId,
              geometry: { type: 'Point', coordinates: [f.longitude, f.latitude] },
              properties: { userId: f.userId },
            })),
          }}
          cluster
          clusterRadius={40}
          onPress={onFriendPress}
        >
          <Layer
            id="friend-clusters"
            type="circle"
            filter={['has', 'point_count']}
            paint={{
              'circle-radius': 18,
              'circle-color': colors.accent,
              'circle-opacity': 0.85,
            }}
          />
          <Layer
            id="friend-cluster-count"
            type="symbol"
            filter={['has', 'point_count']}
            layout={{
              'text-field': '{point_count}',
              'text-size': 13,
            }}
            paint={{
              'text-color': '#fff',
            }}
          />
          <Layer
            id="friend-points"
            type="circle"
            filter={['!', ['has', 'point_count']]}
            paint={{
              'circle-radius': 8,
              'circle-color': colors.accent,
              'circle-stroke-width': 2,
              'circle-stroke-color': colors.background,
            }}
          />
        </GeoJSONSource>
      </Map>

      {selected && (
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
          <Text style={{ color: colors.text, fontSize: 16, fontWeight: '600' }}>Friend</Text>
          <Text style={{ color: colors.textMuted, marginTop: 4 }}>
            Last seen {new Date(selected.recordedAt).toLocaleTimeString()}
          </Text>
        </View>
      )}
    </View>
  );
}

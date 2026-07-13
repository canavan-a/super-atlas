import { useCallback, useEffect, useState } from 'react';
import { FlatList, Pressable, Switch, Text, TextInput, View } from 'react-native';
import { apiJson } from '../api/client';
import { colors } from '../theme/colors';

type Friend = {
  FriendshipID: string;
  FriendID: string;
  Email: string;
  DisplayName: string;
  Status: 'pending' | 'accepted';
  Sharing: boolean;
  IsIncomingRequest: boolean;
};

export default function FriendsScreen() {
  const [friends, setFriends] = useState<Friend[]>([]);
  const [email, setEmail] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [globalPaused, setGlobalPaused] = useState(false);

  const load = useCallback(async () => {
    const data = await apiJson<Friend[]>('/friends/');
    setFriends(data ?? []);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const sendRequest = async () => {
    setError(null);
    try {
      await apiJson('/friends/', { method: 'POST', body: JSON.stringify({ email: email.trim().toLowerCase() }) });
      setEmail('');
      await load();
    } catch (e: any) {
      setError(e.message ?? 'Could not send request');
    }
  };

  const accept = async (id: string) => {
    await apiJson(`/friends/${id}/accept`, { method: 'POST' });
    await load();
  };

  const remove = async (id: string) => {
    await apiJson(`/friends/${id}`, { method: 'DELETE' });
    await load();
  };

  const setSharing = async (id: string, sharing: boolean) => {
    setFriends((prev) => prev.map((f) => (f.FriendshipID === id ? { ...f, Sharing: sharing } : f)));
    await apiJson(`/friends/${id}/sharing`, { method: 'PUT', body: JSON.stringify({ sharing }) });
  };

  const toggleGlobalPause = async (paused: boolean) => {
    setGlobalPaused(paused);
    await apiJson('/locations/pause', { method: 'PUT', body: JSON.stringify({ paused }) });
  };

  return (
    <View style={{ flex: 1, backgroundColor: colors.background, padding: 20 }}>
      <Text style={{ color: colors.text, fontSize: 24, fontWeight: '600', marginBottom: 16 }}>Friends</Text>

      <View
        style={{
          flexDirection: 'row',
          justifyContent: 'space-between',
          alignItems: 'center',
          backgroundColor: colors.surface,
          borderColor: colors.border,
          borderWidth: 1,
          borderRadius: 12,
          padding: 14,
          marginBottom: 16,
        }}
      >
        <View>
          <Text style={{ color: colors.text, fontWeight: '600' }}>Ghost mode</Text>
          <Text style={{ color: colors.textMuted, fontSize: 12, marginTop: 2 }}>
            Hides your location from everyone temporarily.
          </Text>
        </View>
        <Switch value={globalPaused} onValueChange={toggleGlobalPause} />
      </View>

      <View style={{ flexDirection: 'row', gap: 8, marginBottom: 8 }}>
        <TextInput
          placeholder="Friend's email"
          placeholderTextColor={colors.textMuted}
          autoCapitalize="none"
          keyboardType="email-address"
          value={email}
          onChangeText={setEmail}
          style={{
            flex: 1,
            backgroundColor: colors.surface,
            borderColor: colors.border,
            borderWidth: 1,
            borderRadius: 12,
            paddingHorizontal: 14,
            paddingVertical: 12,
            color: colors.text,
          }}
        />
        <Pressable
          onPress={sendRequest}
          disabled={!email}
          style={{
            backgroundColor: colors.accent,
            borderRadius: 12,
            paddingHorizontal: 18,
            justifyContent: 'center',
            opacity: email ? 1 : 0.5,
          }}
        >
          <Text style={{ color: '#fff', fontWeight: '600' }}>Add</Text>
        </Pressable>
      </View>
      {error && <Text style={{ color: colors.danger, marginBottom: 8 }}>{error}</Text>}

      <FlatList
        data={friends}
        keyExtractor={(item) => item.FriendshipID}
        renderItem={({ item }) => (
          <View
            style={{
              backgroundColor: colors.surface,
              borderColor: colors.border,
              borderWidth: 1,
              borderRadius: 12,
              padding: 14,
              marginBottom: 10,
            }}
          >
            <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
              <View>
                <Text style={{ color: colors.text, fontWeight: '600' }}>{item.DisplayName}</Text>
                <Text style={{ color: colors.textMuted, fontSize: 12 }}>{item.Email}</Text>
              </View>
              {item.Status === 'accepted' && (
                <Switch value={item.Sharing} onValueChange={(v) => setSharing(item.FriendshipID, v)} />
              )}
            </View>

            {item.Status === 'pending' && item.IsIncomingRequest && (
              <View style={{ flexDirection: 'row', gap: 8, marginTop: 10 }}>
                <Pressable
                  onPress={() => accept(item.FriendshipID)}
                  style={{ backgroundColor: colors.accent, borderRadius: 8, paddingVertical: 8, paddingHorizontal: 14 }}
                >
                  <Text style={{ color: '#fff', fontWeight: '600' }}>Accept</Text>
                </Pressable>
                <Pressable
                  onPress={() => remove(item.FriendshipID)}
                  style={{ backgroundColor: colors.surfaceElevated, borderRadius: 8, paddingVertical: 8, paddingHorizontal: 14 }}
                >
                  <Text style={{ color: colors.textMuted }}>Decline</Text>
                </Pressable>
              </View>
            )}
            {item.Status === 'pending' && !item.IsIncomingRequest && (
              <Text style={{ color: colors.textMuted, fontSize: 12, marginTop: 8 }}>Request sent — waiting</Text>
            )}
            {item.Status === 'accepted' && (
              <Pressable onPress={() => remove(item.FriendshipID)} style={{ marginTop: 8 }}>
                <Text style={{ color: colors.danger, fontSize: 12 }}>Remove friend</Text>
              </Pressable>
            )}
          </View>
        )}
        ListEmptyComponent={<Text style={{ color: colors.textMuted, marginTop: 20 }}>No friends yet.</Text>}
      />
    </View>
  );
}

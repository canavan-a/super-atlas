import { Pressable, Text, View } from 'react-native';
import { PermissionsState } from '../location/usePermissionsGate';
import { colors } from '../theme/colors';

// Blocks all app usage until location (foreground + background) and
// notification permissions are granted, per project requirement. Explains
// *why* before triggering the OS prompt, which both Apple and Google expect
// for background location and improves grant rates.
export default function PermissionsGateScreen({ permissions }: { permissions: PermissionsState }) {
  const items: { label: string; granted: boolean; detail: string }[] = [
    {
      label: 'Location',
      granted: permissions.foregroundLocationGranted,
      detail: 'So friends can see where you are on the map.',
    },
    {
      label: 'Background location',
      granted: permissions.backgroundLocationGranted,
      detail: 'So your location keeps updating occasionally when the app is closed.',
    },
    {
      label: 'Notifications',
      granted: permissions.notificationsGranted,
      detail: 'So you know when a friend request or update comes in.',
    },
  ];

  return (
    <View style={{ flex: 1, backgroundColor: colors.background, padding: 24, justifyContent: 'center' }}>
      <Text style={{ color: colors.text, fontSize: 26, fontWeight: '600', marginBottom: 8 }}>
        A couple of permissions
      </Text>
      <Text style={{ color: colors.textMuted, fontSize: 15, marginBottom: 28 }}>
        Open Location only works if you grant these. Nothing is shared with anyone until you add a friend.
      </Text>

      {items.map((item) => (
        <View
          key={item.label}
          style={{
            backgroundColor: colors.surface,
            borderColor: colors.border,
            borderWidth: 1,
            borderRadius: 12,
            padding: 16,
            marginBottom: 12,
          }}
        >
          <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text style={{ color: colors.text, fontSize: 16, fontWeight: '600' }}>{item.label}</Text>
            <View
              style={{
                width: 10,
                height: 10,
                borderRadius: 5,
                backgroundColor: item.granted ? colors.success : colors.border,
              }}
            />
          </View>
          <Text style={{ color: colors.textMuted, marginTop: 4, fontSize: 13 }}>{item.detail}</Text>
        </View>
      ))}

      <Pressable
        onPress={permissions.requestAll}
        style={{
          backgroundColor: colors.accent,
          borderRadius: 12,
          paddingVertical: 14,
          marginTop: 12,
          alignItems: 'center',
        }}
      >
        <Text style={{ color: '#fff', fontWeight: '600' }}>
          {permissions.allGranted ? 'All set' : 'Continue'}
        </Text>
      </Pressable>
    </View>
  );
}

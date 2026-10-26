import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { useEffect } from 'react';
import { ActivityIndicator, Pressable, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { Icon } from '../components/Icon';
import { ensureBackgroundLocationTask } from '../location/backgroundTask';
import { PermissionsContext } from '../location/permissionsContext';
import { usePermissionsGate } from '../location/usePermissionsGate';
import { splashHold } from '../logo/splashHold';
import FriendsScreen from '../screens/FriendsScreen';
import MapScreen from '../screens/MapScreen';
import PermissionsGateScreen from '../screens/PermissionsGateScreen';
import PrivacyPolicy from '../privacy/PrivacyPolicy';
import SettingsScreen from '../screens/SettingsScreen';
import SignInScreen from '../screens/SignInScreen';
import { useTheme } from '../theme';

const Stack = createNativeStackNavigator();

// Six screens in all: Sign in, Permissions, Map (home), Friends, Settings,
// and the privacy policy (from Settings).
export default function RootNavigator() {
  const theme = useTheme();
  const { status } = useAuth();
  const permissions = usePermissionsGate();

  useEffect(() => {
    if (status === 'signedIn' && permissions.backgroundLocationGranted) {
      ensureBackgroundLocationTask().catch(() => {});
    }
  }, [status, permissions.backgroundLocationGranted]);

  const loading = status === 'loading' || permissions.loading;
  useEffect(() => {
    if (loading) splashHold.hold();
    else splashHold.release();
  }, [loading]);

  if (loading) {
    return (
      <View style={{ flex: 1, backgroundColor: theme.bg, alignItems: 'center', justifyContent: 'center' }}>
        <ActivityIndicator color={theme.primary} />
      </View>
    );
  }

  const screenOptions = {
    headerStyle: { backgroundColor: theme.bg },
    headerTintColor: theme.text,
    headerShadowVisible: false,
    contentStyle: { backgroundColor: theme.bg },
  };

  if (status === 'signedOut') {
    return (
      <Stack.Navigator screenOptions={{ ...screenOptions, headerShown: false }}>
        <Stack.Screen name="SignIn" component={SignInScreen} />
      </Stack.Navigator>
    );
  }

  if (!permissions.foregroundLocationGranted) {
    return (
      <Stack.Navigator screenOptions={{ ...screenOptions, headerShown: false }}>
        <Stack.Screen name="Permissions">{() => <PermissionsGateScreen permissions={permissions} />}</Stack.Screen>
      </Stack.Navigator>
    );
  }

  return (
    <PermissionsContext.Provider value={permissions}>
      <Stack.Navigator screenOptions={screenOptions}>
        <Stack.Screen name="Map" component={MapScreen} options={{ headerShown: false }} />
        <Stack.Screen
          name="Friends"
          component={FriendsScreen}
          options={({ navigation }) => ({
            headerRight: () => (
              <Pressable onPress={() => navigation.navigate('Settings')} hitSlop={10} accessibilityLabel="Settings">
                <Icon name="gear" color={theme.text} size={22} />
              </Pressable>
            ),
          })}
        />
        <Stack.Screen name="Settings" component={SettingsScreen} />
        <Stack.Screen name="Privacy" component={PrivacyPolicy} options={{ title: 'Privacy policy' }} />
      </Stack.Navigator>
    </PermissionsContext.Provider>
  );
}

import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ActivityIndicator, Pressable, Text, View } from 'react-native';
import { useAuth } from '../auth/AuthContext';
import { usePermissionsGate } from '../location/usePermissionsGate';
import ForgotPasswordScreen from '../screens/ForgotPasswordScreen';
import FriendsScreen from '../screens/FriendsScreen';
import LoginScreen from '../screens/LoginScreen';
import MapScreen from '../screens/MapScreen';
import PermissionsGateScreen from '../screens/PermissionsGateScreen';
import SignupScreen from '../screens/SignupScreen';
import { colors } from '../theme/colors';

const Stack = createNativeStackNavigator();

export default function RootNavigator() {
  const { status, logout } = useAuth();
  const permissions = usePermissionsGate();

  if (status === 'loading' || permissions.loading) {
    return (
      <View style={{ flex: 1, backgroundColor: colors.background, alignItems: 'center', justifyContent: 'center' }}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  const screenOptions = {
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    contentStyle: { backgroundColor: colors.background },
  };

  if (status === 'signedOut') {
    return (
      <Stack.Navigator screenOptions={{ ...screenOptions, headerShown: false }}>
        <Stack.Screen name="Login" component={LoginScreen} />
        <Stack.Screen name="Signup" component={SignupScreen} />
        <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      </Stack.Navigator>
    );
  }

  if (!permissions.allGranted) {
    return (
      <Stack.Navigator screenOptions={{ ...screenOptions, headerShown: false }}>
        <Stack.Screen name="Permissions">{() => <PermissionsGateScreen permissions={permissions} />}</Stack.Screen>
      </Stack.Navigator>
    );
  }

  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen
        name="Map"
        component={MapScreen}
        options={({ navigation }) => ({
          title: 'Open Location',
          headerRight: () => (
            <Pressable onPress={() => navigation.navigate('Friends')} hitSlop={8}>
              <Text style={{ color: colors.accent, fontWeight: '600' }}>Friends</Text>
            </Pressable>
          ),
        })}
      />
      <Stack.Screen
        name="Friends"
        component={FriendsScreen}
        options={{
          headerRight: () => (
            <Pressable onPress={() => logout()} hitSlop={8}>
              <Text style={{ color: colors.danger, fontWeight: '600' }}>Sign out</Text>
            </Pressable>
          ),
        }}
      />
    </Stack.Navigator>
  );
}

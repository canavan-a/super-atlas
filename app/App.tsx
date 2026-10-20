import { DarkTheme, DefaultTheme, NavigationContainer } from '@react-navigation/native';
import { StatusBar } from 'expo-status-bar';
import { useMemo } from 'react';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { useThemedAppIcon } from './src/appIcon';
import { AuthProvider } from './src/auth/AuthContext';
import { SplashGate } from './src/logo/SplashGate';
import { InvitePrompt } from './src/friends/InvitePrompt';
import RootNavigator from './src/navigation/RootNavigator';
import { ThemeProvider, useTheme } from './src/theme';

function ThemedApp() {
  const theme = useTheme();
  useThemedAppIcon();
  const navTheme = useMemo(() => {
    const base = theme.isDark ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: theme.primary,
        background: theme.bg,
        card: theme.bg,
        text: theme.text,
        border: theme.border,
        notification: theme.danger,
      },
    };
  }, [theme]);

  return (
    <SplashGate>
      <NavigationContainer theme={navTheme}>
        <RootNavigator />
        <InvitePrompt />
        <StatusBar style={theme.isDark ? 'light' : 'dark'} />
      </NavigationContainer>
    </SplashGate>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <AuthProvider>
          <ThemedApp />
        </AuthProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}

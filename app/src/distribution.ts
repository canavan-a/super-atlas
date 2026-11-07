// Which build this is (android/app/build.gradle productFlavors, and the iOS
// app), as in super-chat:
//  sideload  distributed outside Google Play (release/release.sh): the
//            in-app updater and opt-in background notifications.
//  play      Google Play: notifications through Firebase instead.
//  appstore  the iOS app (App Store / TestFlight, .github/workflows/ios.yml):
//            notifications through Firebase, like play.
//  web       the browser build: view only, never shares a location.
import { NativeModules, Platform } from 'react-native';

export type Distribution = 'sideload' | 'play' | 'appstore' | 'web';

const flavor = NativeModules.Distribution?.flavor as string | undefined;

export const distribution: Distribution =
  Platform.OS === 'web'
    ? 'web'
    : Platform.OS === 'ios'
      ? 'appstore'
      : flavor === 'play'
        ? 'play'
        : 'sideload';

export const isSideload = distribution === 'sideload';

// The server this build talks to. Native builds have it baked in: Gradle's
// -PapiUrl on Android, the SUPERATLAS_API_URL build setting on iOS (both
// through DistributionModule). The web build is served by that server, so
// it's the page's own origin (the Vite dev server proxies the API, see
// vite.config.ts). The fallbacks reach a dev server on this computer from
// the Android emulator and the iOS simulator.
export const apiUrl: string =
  Platform.OS === 'web'
    ? window.location.origin
    : (NativeModules.Distribution?.apiUrl as string | undefined) ||
      (Platform.OS === 'ios' ? 'http://localhost:8080' : 'http://10.0.2.2:8080');

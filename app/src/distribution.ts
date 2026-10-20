// Which build this is (android/app/build.gradle productFlavors), as in
// super-chat:
//  sideload  distributed outside Google Play (release/release.sh): the
//            in-app updater and opt-in background notifications.
//  play      Google Play: notifications through Firebase instead.
//  web       the browser build: view only, never shares a location.
import { NativeModules, Platform } from 'react-native';

export type Distribution = 'sideload' | 'play' | 'web';

export const distribution: Distribution =
  Platform.OS === 'web' ? 'web' : NativeModules.Distribution?.flavor === 'play' ? 'play' : 'sideload';

export const isSideload = distribution === 'sideload';

// The server this build talks to. Android builds have it baked in by Gradle
// (-PapiUrl); the web build is served by that server, so it's the page's own
// origin (the Vite dev server proxies the API, see vite.config.ts).
export const apiUrl: string =
  Platform.OS === 'web'
    ? window.location.origin
    : (NativeModules.Distribution?.apiUrl as string | undefined) ?? 'http://10.0.2.2:8080';

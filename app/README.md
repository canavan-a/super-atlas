# Super Atlas app

Bare React Native (Android) plus a Vite/react-native-web build from the same
code, built the way super-chat is: a committed `android/` project, the React
Native CLI and Gradle, no Expo build tooling. It does use Expo SDK libraries
(`expo-location`, `expo-background-task`, `expo-secure-store`, `expo-linking`),
linked through Expo's autolinking in `android/settings.gradle`.

```sh
npm ci
npm start                  # Metro
npm run android            # sideload flavor on a device/emulator (system Android SDK)
npm run android:play       # Google Play flavor
npm run web                # browser client on :5173 (proxies the API to :8080)
npm run build:web          # dist/, served by the Go server (SUPERATLAS_WEB_DIR)
```

The Android build talks to the server it was built for: `-PapiUrl`, by default
`http://10.0.2.2:8080` (the dev server, seen from an emulator). For a phone on
your network: `npm run android -- --extra-params -PapiUrl=http://192.168.1.20:8080`.

## Builds

`src/distribution.ts` says which one is running:

- **sideload** (`android/app/src/sideload`): the in-app updater
  (`AppUpdaterModule.kt`, GitHub Releases) and opt-in background notifications
  (`NotifyService.kt`, a foreground service holding the server's
  `/notifications/stream` socket), switched from Settings.
- **play** (`android/app/src/play`): Firebase Cloud Messaging
  (`PushModule.kt`); fill in `res/values/google_services.xml` first.
- **web**: view only. Files ending `.web.ts(x)` replace their native siblings:
  the map is `MapScreen.web.tsx` on maplibre-gl, the location stream only
  receives, and nothing asks for location permission.

## Layout

- `src/screens/` — the five screens: sign in, permissions, map, friends, settings.
- `src/location/` — the live location socket, the background upload task and
  the permission gate.
- `src/api/` — requests with the session token (secure store on Android,
  localStorage on the web).
- `src/theme.tsx` — super-badger's five palettes, each with a matching basemap.
- `src/logo/` — the launch splash, from super-chat.

## Launch splash and icons

The art is generated, not drawn here. Edit `../art/mark.txt` or
`../art/word.txt`, then:

    cd ../tools && go run ./appart              # -> src/logo/logoData.ts, launcher icons, public/ icons
    cd ../tools && go run ./appart -preview /tmp/out   # also PNG previews per theme

The launcher icon follows the theme (`src/appIcon.ts`, `AppIconModule.kt`, one
`<activity-alias>` per theme in the manifest), switching as the app goes to the
background.

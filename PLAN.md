# Super Atlas: move to super-chat's build system, three targets

## Context
Already done and committed under the earlier approved plan:
- the rename
- email-code sign-in with SQLite sessions
- simple friends with invite links, and the UX simplification
- super-badger themes
- the pixel-art logo and splash (commits `33cf7e1`, `6add094`, `cb3f460`)

The user now wants to **stop using Expo as the build system** and build like `../super-chat`:
- a committed `app/android/` project
- the React Native CLI, Metro, Gradle product flavors
- a Vite + react-native-web web build
- the `release/` scripts

Expo **SDK libraries stay** (`expo-location`, `expo-background-task`, `expo-task-manager`, `expo-secure-store`, `expo-linking`, `expo-status-bar`). There's no `expo prebuild`/CNG, no `app.config.js`, and no config plugins.

There are three targets:

1. **sideload** Android (release.sh + GitHub Releases):
   - in-app updater
   - opt-in **background notifications** (asks for the notification permission), via a persistent connection like super-chat's
2. **play** Android (release-play.sh, .aab): notifications through **Firebase Cloud Messaging**, with no updater and no always-on service.
3. **web** (Vite):
   - sign in, see friends on a map, manage friends
   - **never sends a location**
   - says so plainly: "View only — the web version doesn't share your location"

All three share the app's background-location behavior as it is now: the `expo-background-task` periodic upload on Android, and none on web.

The UX stays super simple (5 screens). Notifications cover only two things: "<name> wants to be friends" and "<name> is now your friend".

## Step 0: Housekeeping
- Discard the uncommitted Expo-specific Step 7 work: `app/modules/app-updater`, `app/plugins/withReleaseSigning.js`, and the `app.json`/`package.json` edits.
- Keep `app/src/appUpdater.ts`, the `SettingsScreen` update UI and `release/`; they get adapted below.
- Remove `app/modules/app-icon`, `app/plugins/`, and `app/app.config.js` (their output moves into the committed native project).
- Commit as "drop Expo build config" once Step 1 builds.

## Step 1: Committed bare Android project (super-chat layout)
- Generate `app/android/` **once** with `npx expo prebuild --platform android` from the current config, then commit it and remove `/android` from `app/.gitignore`. That's the standard bare-workflow starting point.
  - It keeps Expo's autolinking in `settings.gradle` and the `MainApplication` wrappers, which the Expo SDK libraries need.
  - From then on it's edited by hand, like super-chat's.
- `app/app.json` becomes RN-CLI style: `{ "name": "SuperAtlasApp", "displayName": "Super Atlas" }`.
- The location permission text and the `expo-location` and `expo-background-task` manifest bits are now written directly in `AndroidManifest.xml`.
- `index.js` uses `AppRegistry.registerComponent(appName, () => App)` like super-chat. The background task module is imported at top level so headless runs define it. Delete `index.ts`.
- Tooling:
  - add `@react-native-community/cli` + `cli-platform-android`, `@react-native/metro-config` and `@react-native/babel-preset`, all matching RN 0.86
  - `metro.config.js` and `babel.config.js` as in super-chat
  - drop `expo-dev-client` and `@expo/ngrok`
- Scripts:
  - `android`: `react-native run-android --mode sideloadDebug`
  - `android:play`: `--mode playDebug`
  - `start`: `react-native start`
  - `web`, `build:web`, `release`, `release:play`
- **Native code in `android/app/src/`**, ported from super-chat's own modules (super-chat's are legacy `ReactPackage` modules, which RN 0.86 still runs through its interop layer):
  - `main/`:
    - `MainActivity` (deep links: `superatlas://invite/*` and `https://<host>/i/*` intent filters; the host comes from a `manifestPlaceholders` value set from `-PapiHost`)
    - `AppIconModule`/`Package`
    - `DistributionModule`/`Package`
    - the 5 `<activity-alias>` launcher entries (from `withAppIcon.js`)
  - `sideload/`:
    - `Flavor.kt`
    - `AppUpdaterModule`/`Package` + FileProvider + `REQUEST_INSTALL_PACKAGES`
    - `NotifyService.kt` (Step 3)
  - `play/`:
    - `Flavor.kt`
    - `PushModule.kt` (FCM token)
    - `res/values/google_services.xml` (template; real values filled in by `release-play.sh`, as super-chat does)
- `build.gradle`:
  - `productFlavors { sideload { targetSdkVersion 34 } play {} }` (the cap reason is the same as super-chat's)
  - firebase-messaging only for `play`
  - release signing from `SUPERATLAS_KEYSTORE_*`
  - `-PversionName/-PversionCode`
- `tools/appart` writes launcher/notification icons straight into `app/android/app/src/main/res`, like super-chat. Drop `app/plugins/res`.

## Step 2: Web build (view only)
- Port super-chat's `vite.config.ts`, `index.html`, `index.web.tsx` and `src/stubs/`. Use react-native-web + the react-native-svg web alias; drop the PWA plugin.
- Add `maplibre-gl` for the web map.
- `.web.tsx`/`.web.ts` variants:

| File | Web behavior |
|---|---|
| `screens/MapScreen.web.tsx` | the same UI on maplibre-gl, plus a fixed "View only — the web version doesn't share your location" banner |
| `location/useLocationStream.web.ts` | receives friends' positions, never sends |
| `location/backgroundTask.web.ts` | no-op |
| `location/usePermissionsGate.web.ts` | always granted |
| `api/client` token store | `localStorage` in place of SecureStore |
| `friends/InvitePrompt` | reads `?invite=<code>` in place of `expo-linking` |
| `appIcon` / `appUpdater` | no-ops |

- On web the API base URL is the page's own origin.
- The server serves the built web app (`SUPERATLAS_WEB_DIR`, at `/` alongside the API), following `../super-chat/server/static`.
- The `/i/<code>` landing page also offers "Open on the web" → `/?invite=<code>`.

## Step 3: Notifications (server + both Android flavors)
**Server:**
- `server/internal/notify/`:
  - `Notify(userID, title, body)` fans out to the user's live notification sockets (a hub like `location.Hub`) and to FCM
  - called on a friend request (to the addressee), on accept (to the requester), and on invite accept (to the inviter), from `friends.Service`
- `server/internal/push/`: port `../super-chat/server/push/push.go`'s FCM HTTP v1 client (OAuth via the service-account JSON), sending to device tokens. A nil client sends nothing.
- Migration `0006_push_tokens.sql`: `push_tokens (token PRIMARY KEY, user_id, session_id → sessions ON DELETE CASCADE, created_at)`. A token is dropped on logout, and when FCM reports it unregistered.
- Routes:
  - `PUT /push/token` `{token}` (authenticated)
  - `GET /notifications/stream`: WebSocket carrying only notify events, never locations
- Config: `SUPERATLAS_FCM_CREDENTIALS_FILE`.

**sideload:**
- A native Kotlin foreground service (`NotifyService.kt`) holds an OkHttp WebSocket to `/notifications/stream`, which is independent of the JS runtime, and posts system notifications. Its ongoing notification reads "Super Atlas — connected for friend requests"; it reconnects with backoff and starts on boot if enabled.
- JS starts and stops it, handing over the session token.
- Settings gets one switch, sideload only: **"Background notifications"**. Turning it on requests `POST_NOTIFICATIONS`. Logging out stops the service.

**play:**
- `PushModule.getToken()` → `PUT /push/token` after sign-in, and again on token refresh.
- FCM "notification" messages are shown by the system with `ic_notification` (manifest `default_notification_icon` meta-data).
- The notification permission is requested once after sign-in.
- No Settings switch.

**web:** none.

## Step 4: Releases
- `release/release.sh` (sideload): as written, minus `expo prebuild`; it runs `./gradlew assembleSideloadRelease -PapiHost=… -PversionName -PversionCode` and attaches `superatlas-app-v*.apk` to the release. `EXPO_PUBLIC_API_URL` is still read from `release/app.env` and baked into the bundle.
- `release/release-play.sh`: port super-chat's, with versionCodes from 100, `.aab` output, and the Firebase values check.
- `release/README.md` documents both, plus Firebase setup:
  - the project + `google_services.xml` values
  - the service-account JSON for the server

## Step 5: Nix (unchanged intent, now with web)
- `flake.nix`: port `../super-chat/flake.nix`, including the Android SDK/JDK in the dev shell; `run-superatlas` and `dev-up` also serve `app/dist` when built.
- `module.nix` (`services.superatlas`):
  - options: `listenAddr`, `dataDir`, `publicOrigin`, `androidCertSha256`, `trustProxy`, `environmentFile` (SMTP), `fcmCredentialsFile`
  - `web.enable` builds the web client via an npm cache, like super-chat's module

## Critical files
- New:
  - `app/android/**` (committed)
  - `app/index.js`, `app/metro.config.js`, `app/babel.config.js`, `app/vite.config.ts`, `app/index.html`, `app/index.web.tsx`
  - `app/src/screens/MapScreen.web.tsx` and the `*.web.ts` variants
  - `server/internal/notify/`, `server/internal/push/`
  - `release/release-play.sh`, `flake.nix`, `module.nix`
- Changed:
  - `app/package.json`, `app/app.json`
  - `app/src/screens/SettingsScreen.tsx`, `app/src/auth/AuthContext.tsx` (push/notify start and stop), `app/src/appIcon.ts` (NativeModules.AppIcon), `app/src/appUpdater.ts` (NativeModules.AppUpdater)
  - `tools/appart/main.go` (output path)
  - `server/internal/friends/service.go`, `server/internal/httpapi/router.go`, `server/cmd/superatlas-server/main.go`
- Removed: `app/app.config.js`, `app/plugins/`, `app/modules/`, `app/index.ts`

## Verification
- **Server:**
  - `go test ./...`
  - new tests:
    - notify fan-out reaches the hub on request, accept and invite accept
    - push tokens are deleted on logout
    - the FCM client against a fake endpoint, as super-chat's `push_test.go` does
  - curl the flow
- **Android:**
  - `cd app/android && ./gradlew assembleSideloadDebug assemblePlayDebug` builds both flavors
  - `npx tsc --noEmit`
  - install sideload on a device or emulator:
    - sign in, see the map, change the theme → the launcher icon follows
    - turn on Background notifications, kill the app, send a friend request from another account → a notification appears
    - Check for updates against a test release
- **Web:**
  - `npm run build:web` and serve it from the Go server
  - sign in, see friends, see the banner
  - the network tab shows no location sends; `/i/<code>` → "Open on the web" adds the friend
- **Releases:** `release.sh` and `release-play.sh` produce a signed APK and AAB (Firebase values are needed for a real play build).

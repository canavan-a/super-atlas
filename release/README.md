# release/

Local signing material and the guided release tools for the Android app,
ported from `../super-badger/release/` and `../super-chat/release/`. There are
two Android builds of the same app (same ID and signing key), as Gradle
flavors in `app/android/app/build.gradle`:

- **sideload** — `release.sh`: a signed APK published as a GitHub Release.
  The app's Settings → Check for updates installs new ones from there, and
  Settings → Background notifications keeps a connection open for friend
  requests.
- **play** — `release-play.sh`: a signed app bundle (`.aab`) for Google Play,
  with notifications through Firebase Cloud Messaging.

(The web build is separate: `npm run build:web` in `app/`, served by the Go
server from `SUPERATLAS_WEB_DIR`.)

Tracked in git: `release.sh`, `release-play.sh`, `gen-keystore.sh`,
`README.md`, `.gitignore`. **Never** committed: `release.keystore`,
`keystore.env`, `app.env`, `play-version.env`, `play-releases.log`, or any
built `.apk`/`.aab`.

## First-time setup

```
./release/gen-keystore.sh
```

Generates `release.keystore` (a real Android signing key, valid ~27 years)
and `keystore.env` (its passwords). Refuses to run again once these exist —
regenerating would produce a *different* key, and Android won't install an
update signed by a different key over an app installed from the old one, so
losing/replacing this keystore means every existing install of the app can
never again be updated in place (only uninstalled and reinstalled fresh).
**Back up `release.keystore` and `keystore.env` somewhere durable outside
this checkout.**

It prints the key's SHA-256 fingerprint. Give that to the server as
`SUPERATLAS_ANDROID_CERT_SHA256` (`services.superatlas.androidCertSha256` in
the NixOS module) so invite links open the app straight away.

The server URL the app is built for goes in `app.env`
(`SUPERATLAS_API_URL=https://atlas.example.com`); `release.sh` asks for it the
first time.

## Cutting a sideload release

```
./release/release.sh
```

(or `npm run release` from `app/`)

The script: checks the toolchain, sources `keystore.env` and `app.env`, lists
existing `app-v*` tags, prompts for the new tag (e.g. `app-v0.3.2`) and a
`versionCode`, runs `./gradlew assembleSideloadRelease` with full output,
then — after a single confirmation — creates and pushes the tag and publishes
a GitHub Release with the signed APK attached (named `superatlas-<tag>.apk`,
matching `app/src/appUpdater.ts`'s `APK_NAME` regex so the in-app updater
actually finds it).

The signed APK is also left at `release/superatlas-<tag>.apk`.

## Google Play build

Once, set up Firebase for notifications:

1. Create a Firebase project and add an Android app with the package
   `com.superatlasapp`.
2. From its `google-services.json`, fill in
   `app/android/app/src/play/res/values/google_services.xml` (the file shows
   which four values go where). They identify the project and ship inside
   the app; they aren't secrets.
3. Project settings → Service accounts → Generate new private key, and give
   that JSON file to the server as `SUPERATLAS_FCM_CREDENTIALS_FILE`
   (`services.superatlas.fcmCredentialsFile`). This one *is* a secret.

Then:

```
./release/release-play.sh
```

(or `npm run release:play` from `app/`). It suggests the next version from the
last Play build (`play-version.env`, with every build logged in
`play-releases.log`), builds and signs `superatlas-play-<version>.aab` here,
and only records the version once the build worked. Play's versionCodes start
at 100, above the sideload builds', so the two never collide. Upload the
`.aab` in the Play Console (Testing → Internal testing is the place to start).

Assumes `git`, `java`, `node`, the Android SDK, and `gh` (authenticated, for
sideload releases). `nix develop` provides node and gh; the JDK and Android
SDK come from the system, as for super-chat.

## iOS (App Store / TestFlight)

The iOS app is `app/ios/`, a committed Xcode project edited by hand like
`android/` (Expo 57's bare template, the React Native CLI, no prebuild).
Builds run on GitHub's macOS runners, not here: pushing a tag like
`ios-v0.1.0` runs `.github/workflows/ios.yml`, which archives, signs and
uploads the build to App Store Connect, where it shows up under TestFlight.
The server URL comes from `app.env`, as for Android.

Once, on the Apple side (team `4W9H62B6C2`, the same as super-chat):

1. Certificates, IDs & Profiles → Identifiers: an App ID `com.superatlasapp`
   with **Push Notifications** and **Associated Domains**.
2. Profiles: an App Store profile for it named **Super Atlas iOS App Store**
   (the Release config looks for that name), using the team's Apple
   Distribution certificate (super-chat's will do).
3. App Store Connect: a new app with that bundle ID.
4. Repo secrets, as in super-chat (`ios.yml`'s header lists them). The
   certificate and API key can be super-chat's; the profile is the new one.
5. The server: `services.superatlas.appleAppIds = [ "4W9H62B6C2.com.superatlasapp" ];`
   so invite links open the app (`/.well-known/apple-app-site-association`).

Notifications (optional, the app works without them): in the Firebase
project, add an iOS app with bundle ID `com.superatlasapp`, upload an APNs
key (Apple: Keys → a key with APNs) under Cloud Messaging, and commit its
`GoogleService-Info.plist` as `app/ios/SuperAtlasApp/GoogleService-Info.plist`.
Like the Android values it isn't a secret. The server needs nothing new: it
already sends through FCM.

### App Review and background location

Apple reviews background location closely. What the app does to pass, and
what to keep in step if it changes:

- **Purpose strings** (`Info.plist`): what's shared, who sees it, that it
  continues with the app closed, and what Motion & Fitness is for. They
  match `PermissionsGateScreen.tsx` and the privacy policy.
- **Permission flow:** one explainer screen with a single "Continue" button,
  then "While Using", then iOS's "Change to Always Allow" prompt (Apple's
  recommended order). Saying no doesn't lock anyone out: the app opens
  view-only, and a banner points to Settings. Precise Location off gets its
  own banner.
- **Background modes:** only `location` (MotionTracker.m) and `processing`
  (expo-background-task's fallback). No `remote-notification`, since
  notifications are plain alerts. Declaring an unused mode is a common
  rejection (Guideline 2.5.4).
- **Visible while it runs:** iOS's blue location indicator shows while
  tracking, like Android's notification, and tracking stops after a few
  still minutes.
- **Privacy manifest** (`PrivacyInfo.xcprivacy`) and the **App Store privacy
  label** must agree. Answer the label in App Store Connect with:
  - Location → Precise Location
  - Contact Info → Email Address, Name
  - User Content → Photos or Videos (the profile photo)
  - Identifiers → User ID, Device ID (the notification token)

  Each one: linked to the user, used for App Functionality only, not used for
  tracking. Tracking: No. Motion data never leaves the phone, so it isn't
  declared.
- **Account deletion** is in the app (Settings → Delete account), as Apple
  requires.
- **Privacy policy URL:** `https://superatlas.lunargen.com/privacy`.
- **Review notes and demo account:** set `services.superatlas.reviewAccount`
  and give its email and code in App Review Information. It starts with no
  friends, so befriend it from your own account first, so the reviewer sees
  someone on the map. Add notes along
  these lines: "Super Atlas shares your location with friends you've
  accepted. Background location lets them keep seeing you while the app is
  closed: it runs while you're moving, shown by the blue location indicator,
  and stops when you've been still a few minutes. Motion & Fitness is used
  on the device only, to notice that you've stopped. The demo account
  already has a friend, so the map shows someone." A short screen recording of the flow
  helps.

Known limit: if someone swipes the app away in the app switcher, iOS doesn't
relaunch it for location until they open it again. Friends then see their
last position going stale.

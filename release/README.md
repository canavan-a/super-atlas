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

# super-atlas

See your friends on a map. That's it: no groups, no circles, no feeds. It
looks and feels like super-badger and super-chat (the same themes, boot
screen and sideloaded Android APK) and runs on your own server.

- **Friends only.** You see the people you've added, and they see you. Add
  someone with your personal invite link (or their email); either of you can
  remove the other any time.
- **One privacy switch.** Ghost mode hides you from everyone until you turn it
  off. That's the only setting there is.
- **Capped, no passwords.** Anyone can sign up, with a code emailed to them,
  until there are 150 accounts; whoever runs the server raises that with
  `superatlas-admin cap <n>`. A device stays signed in until it's unused for 90 days or
  you sign out.
- **Live, then periodic.** While the app is open your position streams to
  friends; while it's closed Android uploads one about every 15 minutes.
  Positions older than an hour are drawn faded.
- **Four builds of one app:**
  - **Android, sideloaded** (GitHub Releases): updates itself from Settings,
    and can keep a connection open for friend-request notifications.
  - **Android, Google Play**: notifications through Firebase.
  - **iOS** (App Store / TestFlight, built by `.github/workflows/ios.yml` on an
    `ios-vX.Y.Z` tag): notifications through Firebase, sharing on the move.
  - **Web**: view only. It shows your friends but never asks for or shares
    your location, and says so on the map.

## Layout

```
app/        React Native app: Android (committed android/, two Gradle flavors),
            iOS (committed ios/) and a Vite/react-native-web build from the
            same code
server/     Go + SQLite server: email-code sign-in, friends, live locations,
            notifications (WebSocket + FCM), serves the web build
tools/      appart: generates the logo data and launcher icons from art/
art/        the pixel art (a scroll, a world map, and "Super Atlas" in super-badger's lettering)
release/    APK/AAB signing and the guided releases (app-v* tags, Play bundles)
module.nix  NixOS module: services.superatlas
```

## Running it

```sh
nix develop          # prints the dev commands
dev-up               # server on 127.0.0.1:8080 (dev mode: sign-in codes go to the log)
superatlas-admin cap                     # accounts so far, and the cap (150)
run-app-web          # web client on :5173, talking to that server
run-app-android      # on an emulator, which reaches the server at 10.0.2.2:8080
run-tests            # Go tests and the app's typecheck
```

On NixOS:

```nix
services.superatlas = {
  enable = true;
  publicOrigin = "https://atlas.example.com";
  environmentFile = "/run/secrets/superatlas.env";   # SUPERATLAS_SMTP_EMAIL / _APP_PASSWORD
  signupAlertEmail = "me@example.com";               # emailed when someone signs up
  web.enable = true;                                 # the view-only browser client at /
  # fcmCredentialsFile = "/run/secrets/superatlas-fcm.json";  # Google Play notifications
};
```

It listens on loopback; put a Cloudflare tunnel (or any TLS proxy) in front.
Anyone can sign up until there are 150 accounts (NPCs don't count); after
that only existing accounts sign in. `superatlas-admin cap` shows how many
are taken, `cap <n>` changes the cap, `list` shows everyone, and
`remove <email>` deletes an account, freeing its place.

## Releases

`./release/gen-keystore.sh` once, then `./release/release.sh` for the
sideloaded APK (tags `app-vX.Y.Z`, publishes a GitHub release that the app's
Settings → Check for updates installs from) or `./release/release-play.sh` for
the Google Play bundle. Put the key's SHA-256 in
`services.superatlas.androidCertSha256` so invite links open straight in the
app. Details, including Firebase setup, in [`release/README.md`](release/README.md).

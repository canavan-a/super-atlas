# Remaining work

## App (`app/`)
- [ ] Offline map tiles: prefetch/cache a region with MapLibre's offline manager, add a "download this area" action, and optionally auto-refresh cached tiles on wifi. Render cached last-known friend positions on top when there's no connection.
- [x] Resolve friend display names on the map (names now come with every location).
- [x] App icons, splash screen, adaptive icon assets (pixel art from `art/`, one launcher icon per theme).
- [ ] Real device testing: background location task, the sideload notification service across reboots, the in-app updater, and both flavors' notifications. Nothing has run on a device yet.
- [x] Handle a permanently denied permission (the gate's button opens Settings).
- [x] `/me` on launch.
- [x] Notifications on friend requests (sideload: background connection; Play: FCM).
- [ ] Loading/error states polish (friends list, map) and pull-to-refresh.
- [ ] iOS: not set up at all (no `ios/` project).

## Server (`server/`)
- [x] Invite-only sign-in: an email allowlist managed with `superatlas-admin`.
- [x] Rate limiting: per IP on `/auth/*` and invites, per email on codes and guesses, an hourly cap on all codes sent, and per account on signed-in endpoints.
- [ ] Keep session tokens out of the request log (the WebSocket streams pass them as `?token=`).
- [ ] Size limits on request bodies and WebSocket messages, and server timeouts.
- [ ] Build the web client without npm install scripts running as root (`module.nix`).
- [x] FCM dispatch on friend events.
- [x] `/me` endpoint.
- [ ] Structured logging / error tracking (currently just chi's default request logger).
- [ ] SQLite backups (single-file DB, so backup = copy the file + WAL).
- [x] Go test suite (auth, friends, locations, notifications, push).

## Store shipping
- [ ] Privacy policy page (required by Google Play for a precise-location app); could be a page of the web build, as super-chat's `/privacy`.
- [ ] Firebase project + `google_services.xml` values + the server's service-account key (see `release/README.md`).
- [ ] Google Play: dev account, Sensitive App Permissions declaration for background location, prominent in-app disclosure before requesting "Allow all the time".

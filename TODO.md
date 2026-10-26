# Remaining work

## Deploy (the superchat host, `/etc/nixos`)
- [x] `services.superatlas` in `modules/superchat.nix` (127.0.0.1:8090, `https://superatlas.lunargen.com`, web client on) and the `super-atlas` flake input.
- [ ] Push this repo, then `nix flake update super-atlas` in `/etc/nixos` so the host picks up the allowlist.
- [ ] Put the mail login on the box: `/etc/superatlas/env` with `SUPERATLAS_SMTP_EMAIL` and `SUPERATLAS_SMTP_APP_PASSWORD`, plus `SUPERATLAS_SMTP_HOST`/`_PORT` if not Gmail (e.g. `smtp.zoho.com`/`587`) (the service won't start without it).
- [x] Route `superatlas.lunargen.com` → `127.0.0.1:8090` (through the Super Matrix relay).
- [ ] Deploy, check port 8090 is free first (`sudo ss -ltnp | grep 8090`), then `superatlas-admin allow <email>` for each person.
- [x] The release key's SHA-256 in `androidCertSha256` so invite links open the app.
- [ ] After the first Play upload, add Play's app signing key SHA-256 (Play Console → App integrity → App signing) to `androidCertSha256` too.
- [x] The Firebase service account key on the box at `/etc/superatlas/fcm.json` (the service won't start without it now that `fcmCredentialsFile` is set).
- [ ] Optional: `inputs.nixpkgs.follows = "nixpkgs"` on the `super-atlas` input to skip its second nixpkgs.

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
- [x] FCM dispatch on friend events.
- [x] `/me` endpoint.
- [x] Go test suite (auth, friends, locations, notifications, push, allowlist).
- [x] Configurable SMTP host and port (`SUPERATLAS_SMTP_HOST`/`_PORT`, Gmail by default), e.g. Zoho or a relay on your own domain.
- [ ] SQLite backups (single-file DB, so backup = copy the file + WAL).
- [ ] Structured logging / error tracking (currently just chi's default request logger).

### Hardening (left over from the public-exposure audit)
- [ ] Keep session tokens out of the request log (the WebSocket streams pass them as `?token=`).
- [ ] Size limits on request bodies and WebSocket messages, and server timeouts.
- [ ] Build the web client without npm install scripts running as root (`module.nix`; super-chat's module has the same issue).
- [ ] Range-check shared locations (latitude/longitude, `recordedAt` not in the future).
- [ ] Don't let a push token move to another account (`PUT /push/token` reassigns one that already exists).

## Store shipping
- [ ] Privacy policy page (required by Google Play for a precise-location app); could be a page of the web build, as super-chat's `/privacy`.
- [ ] Firebase project + `google_services.xml` values + the server's service-account key (see `release/README.md`).
- [ ] Google Play: dev account, Sensitive App Permissions declaration for background location, prominent in-app disclosure before requesting "Allow all the time".

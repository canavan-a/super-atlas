# Remaining work

## App (`app/`)
- [ ] Offline map tiles: prefetch/cache a region with MapLibre's offline manager, add a "download this area" action, and optionally auto-refresh cached tiles on wifi. Render cached last-known friend positions on top when there's no connection.
- [ ] Resolve friend display name/avatar on the map (marker tap currently just shows "Friend" + last-seen time — needs to cross-reference the friends list by `userId`).
- [ ] App icons, splash screen, adaptive icon assets (currently placeholder Expo defaults).
- [x] Link project to EAS (`extra.eas.projectId` set in `app.json`).
- [ ] Fill in EAS Build/Submit config (`eas.json`) for iOS + Android.
- [ ] Real device testing of background location task (`expo-background-task`) — simulators don't reliably exercise OS background scheduling.
- [ ] Handle the case where a user permanently denies a permission (Settings deep-link prompt) instead of just re-showing the request button.
- [ ] `/me` endpoint + call it on app launch to validate the stored access token belongs to a real, still-existing account (currently we just check "is there a token").
- [ ] Push notifications: register device token with server, send on incoming friend request (`expo-notifications` permission is requested but nothing sends a push yet).
- [ ] Loading/error states polish (friends list, map) and pull-to-refresh.

## Server (`server/`)
- [ ] Rate limiting on `/auth/*` (signup/login) to blunt brute-force and enumeration.
- [ ] Push notification dispatch (APNs/FCM) on friend request events.
- [ ] `/me` endpoint for token validation on app launch.
- [ ] Structured logging / error tracking (currently just chi's default request logger).
- [ ] Config for CORS if a web client is ever added.
- [ ] Deployment: containerize (Dockerfile), decide on host, and set up SQLite file backups (single-file DB, so backup = copy the file + WAL).
- [ ] Tests: only manually verified so far via curl; no automated Go test suite yet.

## Store shipping
- [ ] Privacy policy page (required by both stores for a precise-location app) — can be a static page in this repo.
- [ ] Apple: App Store Connect account, background-location justification in review notes, privacy nutrition label.
- [ ] Google Play: $25 dev account, Sensitive App Permissions declaration, prominent in-app disclosure before requesting "Allow all the time".
- [ ] EAS Build + EAS Submit pipeline for both platforms.


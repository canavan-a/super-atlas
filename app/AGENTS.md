# Building this app

This app builds like ../super-chat: committed `android/` and `ios/` projects
edited by hand, the React Native CLI, Gradle flavors (`sideload`, `play`), an
iOS app built on GitHub's macOS runners (`.github/workflows/ios.yml`) and a
Vite web build. Don't run `expo prebuild`, add config plugins, or use EAS:
there is no `app.config.js` and neither native folder is generated. What a
library's config plugin would have done (Info.plist keys, Podfile hooks such
as MapLibre's `$MLRN.post_install`) is written into `ios/` by hand.

It does use Expo SDK libraries (expo-location, expo-background-task,
expo-secure-store, expo-linking). Expo HAS CHANGED: read the exact versioned
docs at https://docs.expo.dev/versions/v57.0.0/ before writing code against
them.

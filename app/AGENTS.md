# Building this app

This app builds like ../super-chat: a committed `android/` project edited by
hand, the React Native CLI, Gradle flavors (`sideload`, `play`) and a Vite web
build. Don't run `expo prebuild`, add config plugins, or use EAS: there is no
`app.config.js` and `android/` is not generated.

It does use Expo SDK libraries (expo-location, expo-background-task,
expo-secure-store, expo-linking). Expo HAS CHANGED: read the exact versioned
docs at https://docs.expo.dev/versions/v57.0.0/ before writing code against
them.

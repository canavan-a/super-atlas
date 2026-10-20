import path from 'path';

import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

// The web build, ported from super-chat's: aliases "react-native" to
// react-native-web so the same src/ renders in a browser. The web app is view
// only (it never sends a location); where it differs from Android it uses a
// `.web.ts(x)` file next to the native one, e.g. MapScreen.web.tsx on
// maplibre-gl. The Go server serves the built app (SUPERATLAS_WEB_DIR), so in
// production the API is the page's own origin; `npm run web` proxies it to a
// dev server on :8080.
const API_PATHS = ['/auth', '/me', '/friends', '/invites', '/locations', '/push', '/notifications', '/i/', '/healthz'];

export default defineConfig(({mode}) => ({
  plugins: [react({include: /\.(jsx|js|tsx|ts)$/})],
  server: {
    proxy: Object.fromEntries(
      API_PATHS.map(p => [p, {target: 'http://localhost:8080', ws: p === '/locations' || p === '/notifications'}]),
    ),
  },
  resolve: {
    alias: {
      // More specific "react-native/..." aliases must precede the generic
      // "react-native" one below: Vite matches alias keys in order.
      //
      // react-native-svg ships a DOM-based web implementation, but its
      // package entry points at the native Fabric components; aliased
      // straight to the web elements (see super-chat's vite.config.ts for
      // the long story).
      'react-native-svg': path.resolve(__dirname, 'node_modules/react-native-svg/lib/module/elements.web.js'),
      'react-native/Libraries/Utilities/codegenNativeComponent': path.resolve(__dirname, 'src/stubs/codegenNativeComponentWebStub.ts'),
      // react-native-web plus the TurboModuleRegistry react-native-svg wants.
      'react-native': path.resolve(__dirname, 'src/stubs/reactNativeWebShim.ts'),
      // Flow-syntax-only package react-native-svg pulls in even on web.
      '@react-native/assets-registry/registry': path.resolve(__dirname, 'src/stubs/assetsRegistryWebStub.ts'),
    },
    extensions: ['.web.tsx', '.web.ts', '.web.jsx', '.web.js', '.tsx', '.ts', '.jsx', '.js'],
  },
  define: {
    // react-native-web checks this at runtime.
    'process.env.NODE_ENV': JSON.stringify(mode === 'production' ? 'production' : 'development'),
    // React Native's `global`, which react-native-web's Animated still calls.
    global: 'globalThis',
  },
}));

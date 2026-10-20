// Package static serves the app's built web client (`npm run build:web`, a
// Vite/react-native-web bundle with nothing secret in it) at /, next to the
// API, adapted from super-chat's server/static. Being the same origin lets the
// web client call the API without CORS.
package static

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Handler serves dir as static files with SPA fallback: a path that doesn't
// exist on disk gets dir/index.html, so the client's own routing survives a
// refresh or deep link. Vite's content-hashed assets are cached for good;
// everything else is revalidated so updates show up.
func Handler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}

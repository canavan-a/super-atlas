package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/go-chi/chi/v5"
)

const (
	androidPackage = "com.superatlasapp"
	latestRelease  = "https://github.com/canavan-a/super-atlas/releases/latest"
)

// inviteURLFunc builds shareable invite links on publicOrigin, or on the
// request's own host when that isn't configured (local dev).
func inviteURLFunc(publicOrigin string) func(r *http.Request, code string) string {
	return func(r *http.Request, code string) string {
		origin := publicOrigin
		if origin == "" {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			origin = scheme + "://" + r.Host
		}
		return origin + "/i/" + code
	}
}

// landing is what an invite link opens when the app isn't installed or the
// link wasn't routed to it.
func landing(svc *friends.Service, hasWeb bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		name, err := svc.InviteOwner(r.Context(), code)
		if err != nil {
			name = ""
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		_ = landingTmpl.Execute(w, map[string]any{
			"Name":     name,
			"AppLink":  template.URL("superatlas://invite/" + code),
			"Download": latestRelease,
			"Web":      hasWeb,
			"WebLink":  "/?invite=" + url.QueryEscape(code),
		})
	}
}

var landingTmpl = template.Must(template.New("landing").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>Super Atlas invite</title>
<style>
  :root { color-scheme: dark; }
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#0d1117; color:#e6edf3; font:15px/1.5 system-ui, sans-serif; }
  main { max-width:420px; padding:24px; text-align:center; }
  h1 { font-size:20px; margin:0 0 8px; }
  p { color:#8b949e; margin:0 0 20px; }
  a { display:block; padding:12px; border-radius:10px; border:1px solid #30363d; background:#2f81f7;
      color:#fff; font-weight:600; text-decoration:none; margin-bottom:10px; }
  a.secondary { background:#161b22; color:#e6edf3; }
</style></head>
<body><main>
{{if .Name}}
  <h1>{{.Name}} wants to be friends on Super Atlas</h1>
  <p>Open this in the app to add them. You'll see each other on the map.</p>
  <a href="{{.AppLink}}">Open in Super Atlas</a>
  <a class="secondary" href="{{.Download}}">Get the app</a>
  {{if .Web}}<a class="secondary" href="{{.WebLink}}">Open on the web</a>{{end}}
{{else}}
  <h1>This invite link doesn't work anymore</h1>
  <p>Ask your friend to send you a new one.</p>
{{end}}
</main></body></html>`))

// assetLinks lets Android open /i/ links straight in the app, once the
// release signing cert's fingerprint is configured.
func assetLinks(certSHA256 []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if len(certSHA256) == 0 {
			_, _ = w.Write([]byte("[]"))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"relation": []string{"delegate_permission/common.handle_all_urls"},
			"target": map[string]any{
				"namespace":                "android_app",
				"package_name":             androidPackage,
				"sha256_cert_fingerprints": certSHA256,
			},
		}})
	}
}

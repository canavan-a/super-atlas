package httpapi

import (
	"net/http"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/config"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/canavan-a/super-atlas/server/internal/notify"
	"github.com/canavan-a/super-atlas/server/internal/ratelimit"
	"github.com/canavan-a/super-atlas/server/internal/static"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	Config           *config.Config
	AuthService      *auth.Service
	AuthHandlers     *auth.Handlers
	FriendService    *friends.Service
	FriendHandlers   *friends.Handlers
	LocationHandlers *location.Handlers
	NotifyHandlers   *notify.Handlers
}

// InviteURL is the link builder friends.Handlers needs; exposed so main can
// construct the handlers before the router.
func InviteURL(cfg *config.Config) func(r *http.Request, code string) string {
	return inviteURLFunc(cfg.PublicOrigin)
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	byIP := ratelimit.IPKeyer(d.Config.TrustProxy)
	limit := func(n int, per time.Duration) func(http.Handler) http.Handler {
		return ratelimit.Middleware(ratelimit.New(n, per), byIP)
	}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.With(limit(60, time.Minute)).Get("/i/{code}", landing(d.FriendService, d.Config.WebDir != ""))
	r.Get("/.well-known/assetlinks.json", assetLinks(d.Config.AndroidCertSHA256))

	r.Route("/auth", func(r chi.Router) {
		r.With(limit(10, 10*time.Minute)).Post("/code", d.AuthHandlers.Code)
		r.With(limit(30, 10*time.Minute)).Post("/verify", d.AuthHandlers.Verify)
	})
	r.With(limit(60, time.Minute)).Get("/invites/{code}", d.FriendHandlers.InvitePreview)

	r.Group(func(r chi.Router) {
		r.Use(d.AuthService.RequireAuth)

		r.Get("/me", d.AuthHandlers.Me)
		r.Post("/auth/logout", d.AuthHandlers.Logout)

		r.Route("/friends", func(r chi.Router) {
			r.Get("/", d.FriendHandlers.List)
			r.Post("/", d.FriendHandlers.SendRequest)
			r.Get("/invite", d.FriendHandlers.Invite)
			r.Post("/invite/reset", d.FriendHandlers.ResetInvite)
			r.Post("/{id}/accept", d.FriendHandlers.Accept)
			r.Delete("/{id}", d.FriendHandlers.Remove)
		})
		r.With(limit(30, 10*time.Minute)).Post("/invites/{code}/accept", d.FriendHandlers.AcceptInvite)

		r.Put("/push/token", d.NotifyHandlers.PutToken)
		r.Get("/notifications/stream", d.NotifyHandlers.Stream)

		r.Route("/locations", func(r chi.Router) {
			r.Get("/stream", d.LocationHandlers.Stream)
			r.Post("/batch", d.LocationHandlers.BatchIngest)
			r.Get("/snapshot", d.LocationHandlers.Snapshot)
			r.Put("/pause", d.LocationHandlers.SetPaused)
		})
	})

	// The web client, when it's been built and configured.
	if d.Config.WebDir != "" {
		r.NotFound(static.Handler(d.Config.WebDir).ServeHTTP)
	}

	return r
}

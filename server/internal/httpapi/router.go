package httpapi

import (
	"net/http"

	"github.com/canavan-a/open-location/server/internal/auth"
	"github.com/canavan-a/open-location/server/internal/friends"
	"github.com/canavan-a/open-location/server/internal/location"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	AuthService    *auth.Service
	AuthHandlers   *auth.Handlers
	FriendHandlers *friends.Handlers
	LocationHandlers *location.Handlers
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", d.AuthHandlers.Signup)
		r.Post("/login", d.AuthHandlers.Login)
		r.Post("/refresh", d.AuthHandlers.Refresh)
		r.Post("/logout", d.AuthHandlers.Logout)
		r.Post("/forgot-password", d.AuthHandlers.ForgotPassword)
		r.Post("/reset-password", d.AuthHandlers.ResetPassword)
	})

	r.Group(func(r chi.Router) {
		r.Use(d.AuthService.RequireAuth)

		r.Route("/friends", func(r chi.Router) {
			r.Get("/", d.FriendHandlers.List)
			r.Post("/", d.FriendHandlers.SendRequest)
			r.Get("/name/{userId}", d.FriendHandlers.GetName)
			r.Post("/{id}/accept", d.FriendHandlers.Accept)
			r.Delete("/{id}", d.FriendHandlers.Remove)
			r.Put("/{id}/sharing", d.FriendHandlers.SetSharing)
		})

		r.Route("/locations", func(r chi.Router) {
			r.Get("/stream", d.LocationHandlers.Stream)
			r.Post("/batch", d.LocationHandlers.BatchIngest)
			r.Get("/snapshot", d.LocationHandlers.Snapshot)
			r.Put("/pause", d.LocationHandlers.SetPaused)
		})
	})

	return r
}

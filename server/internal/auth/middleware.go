package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const (
	userIDContextKey contextKey = "userID"
	tokenContextKey  contextKey = "token"
)

func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			// WebSocket clients can't set custom headers before the upgrade
			// handshake, so the stream endpoint also accepts the token as a
			// query param.
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		userID, err := s.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrInvalidSession) {
				writeError(w, http.StatusUnauthorized, "signed out")
				return
			}
			writeError(w, http.StatusInternalServerError, "could not check session")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		ctx = context.WithValue(ctx, tokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns the authenticated user's ID, set by RequireAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDContextKey).(string)
	return id, ok
}

func tokenFromContext(ctx context.Context) string {
	t, _ := ctx.Value(tokenContextKey).(string)
	return t
}

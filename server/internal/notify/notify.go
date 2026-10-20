// Package notify tells people about friend events ("<name> wants to be
// friends", "<name> is now your friend"), on whichever channel their device
// listens to: the sideloaded app's background connection (the
// /notifications/stream WebSocket, NotifyService.kt) or Firebase Cloud
// Messaging for the Google Play build. The web build gets neither.
package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/canavan-a/super-atlas/server/internal/push"
	"github.com/gorilla/websocket"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SetToken records an FCM token for a session, moving it over if another
// session had it (the same device, signed in again).
func (s *Store) SetToken(ctx context.Context, token, userID, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO push_tokens (token, user_id, session_id) VALUES (?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET user_id = excluded.user_id, session_id = excluded.session_id
	`, token, userID, sessionID)
	return err
}

func (s *Store) TokensFor(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT token FROM push_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *Store) DeleteToken(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_tokens WHERE token = ?`, token)
	return err
}

// Event is what a notification socket receives.
type Event struct {
	Type  string `json:"type"` // always "notify"
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Service struct {
	store *Store
	hub   *location.Hub
	fcm   *push.FCM
}

// NewService takes a nil fcm when FCM isn't configured.
func NewService(store *Store, fcm *push.FCM) *Service {
	return &Service{store: store, hub: location.NewHub(), fcm: fcm}
}

// Notify tells userID's devices. It never blocks the caller: FCM sends run in
// the background.
func (s *Service) Notify(userID, title, body string) {
	payload, _ := json.Marshal(Event{Type: "notify", Title: title, Body: body})
	s.hub.PushTo(userID, payload)
	if s.fcm == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tokens, err := s.store.TokensFor(ctx, userID)
		if err != nil {
			log.Printf("notify: tokens: %v", err)
			return
		}
		for _, t := range tokens {
			err := s.fcm.Send(ctx, t, title, body)
			if errors.Is(err, push.ErrUnregistered) {
				_ = s.store.DeleteToken(ctx, t)
			} else if err != nil {
				log.Printf("notify: %v", err)
			}
		}
	}()
}

// ---- HTTP ----

type Handlers struct {
	service  *Service
	upgrader websocket.Upgrader
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service: service,
		upgrader: websocket.Upgrader{
			// Native clients only; auth is the session token, not Origin.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

type tokenRequest struct {
	Token string `json:"token"`
}

// PutToken registers this session's device for FCM notifications.
func (h *Handlers) PutToken(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	sessionID, _ := auth.SessionIDFromContext(r.Context())
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" || len(req.Token) > 4096 {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := h.service.store.SetToken(r.Context(), strings.TrimSpace(req.Token), userID, sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Stream is the sideloaded app's background connection: it only ever
// receives notify events, never locations.
func (h *Handlers) Stream(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := location.NewClient(conn)
	h.service.hub.Register(userID, client)
	defer func() {
		h.service.hub.Unregister(userID, client)
		client.Close()
	}()
	go client.WritePump()
	// Nothing comes the other way; reading just notices the close (and
	// answers the client's pings).
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

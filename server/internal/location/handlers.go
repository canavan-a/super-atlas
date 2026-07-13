package location

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/canavan-a/open-location/server/internal/auth"
	"github.com/gorilla/websocket"
)

type Handlers struct {
	service  *Service
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewHandlers(service *Service, hub *Hub) *Handlers {
	return &Handlers{
		service: service,
		hub:     hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Mobile clients connect cross-origin by nature; auth is via the
			// bearer-token handshake below, not Origin checks.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type wsIncoming struct {
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	AccuracyM  *float64 `json:"accuracyM,omitempty"`
	RecordedAt string   `json:"recordedAt"`
}

// Stream upgrades to a WebSocket for foreground, low-latency two-way updates:
// the client sends its own position as it moves, and receives friends'
// updates pushed live from the hub.
func (h *Handlers) Stream(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := NewClient(conn)
	h.hub.Register(userID, client)
	defer func() {
		h.hub.Unregister(userID, client)
		client.Close()
	}()

	go client.WritePump()

	// Send an initial snapshot of friends' last-known positions so the map
	// has something to show before any friend's next live update arrives.
	if snapshot, err := h.service.SnapshotForViewer(r.Context(), userID); err == nil {
		for _, loc := range snapshot {
			payload, _ := json.Marshal(updatePayload{
				Type:       "location_update",
				UserID:     loc.UserID,
				Latitude:   loc.Latitude,
				Longitude:  loc.Longitude,
				AccuracyM:  loc.AccuracyM,
				RecordedAt: loc.RecordedAt.UTC().Format(time.RFC3339),
			})
			client.Send(payload)
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			client.logReadErr(err)
			return
		}
		var in wsIncoming
		if err := json.Unmarshal(msg, &in); err != nil {
			continue
		}
		recordedAt, err := time.Parse(time.RFC3339, in.RecordedAt)
		if err != nil {
			recordedAt = time.Now()
		}
		_ = h.service.UpdateLocation(r.Context(), userID, in.Latitude, in.Longitude, in.AccuracyM, recordedAt)
	}
}

type batchPoint struct {
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	AccuracyM  *float64 `json:"accuracyM,omitempty"`
	RecordedAt string   `json:"recordedAt"`
}

type batchRequest struct {
	Points []batchPoint `json:"points"`
}

// BatchIngest accepts location points collected while the app was backgrounded.
// Only the most recent point is persisted/broadcast; older points in the
// batch exist so the client doesn't lose data queued while offline.
func (h *Handlers) BatchIngest(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	var body batchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Points) == 0 {
		writeError(w, http.StatusBadRequest, "points is required")
		return
	}

	latest := body.Points[0]
	latestTime, _ := time.Parse(time.RFC3339, latest.RecordedAt)
	for _, p := range body.Points[1:] {
		t, err := time.Parse(time.RFC3339, p.RecordedAt)
		if err == nil && t.After(latestTime) {
			latest = p
			latestTime = t
		}
	}
	if latestTime.IsZero() {
		latestTime = time.Now()
	}

	if err := h.service.UpdateLocation(r.Context(), userID, latest.Latitude, latest.Longitude, latest.AccuracyM, latestTime); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save location")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type pauseRequest struct {
	Paused bool `json:"paused"`
}

func (h *Handlers) SetPaused(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body pauseRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.SetPaused(r.Context(), userID, body.Paused); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update sharing state")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Snapshot(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	locs, err := h.service.SnapshotForViewer(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load locations")
		return
	}
	writeJSON(w, http.StatusOK, locs)
}

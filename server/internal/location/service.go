package location

import (
	"context"
	"encoding/json"
	"time"

	"github.com/canavan-a/open-location/server/internal/friends"
)

type Service struct {
	store       *Store
	friendStore *friends.Store
	hub         *Hub
}

func NewService(store *Store, friendStore *friends.Store, hub *Hub) *Service {
	return &Service{store: store, friendStore: friendStore, hub: hub}
}

type updatePayload struct {
	Type       string  `json:"type"`
	UserID     string  `json:"userId"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	AccuracyM  *float64 `json:"accuracyM,omitempty"`
	RecordedAt string  `json:"recordedAt"`
}

// UpdateLocation stores a new position for userID and, unless the user has
// paused sharing, pushes it live to any friend who currently has this user's
// updates visible and an open WebSocket connection.
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64, accuracy *float64, recordedAt time.Time) error {
	if err := s.store.Upsert(ctx, userID, lat, lng, accuracy, recordedAt); err != nil {
		return err
	}

	settings, err := s.store.GetSettings(ctx, userID)
	if err != nil {
		return err
	}
	if settings.SharingPaused {
		return nil
	}

	viewers, err := s.viewersFor(ctx, userID)
	if err != nil {
		return err
	}
	if len(viewers) == 0 {
		return nil
	}

	payload, err := json.Marshal(updatePayload{
		Type:       "location_update",
		UserID:     userID,
		Latitude:   lat,
		Longitude:  lng,
		AccuracyM:  accuracy,
		RecordedAt: recordedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	for _, viewerID := range viewers {
		s.hub.PushTo(viewerID, payload)
	}
	return nil
}

// viewersFor returns the IDs of accepted friends who have sharing enabled
// *toward* userID, i.e. who are allowed to see userID move.
func (s *Service) viewersFor(ctx context.Context, userID string) ([]string, error) {
	views, err := s.friendStore.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var viewers []string
	for _, v := range views {
		if v.Status == "accepted" && v.Sharing {
			viewers = append(viewers, v.FriendID)
		}
	}
	return viewers, nil
}

// SnapshotForViewer returns the last-known locations of userID's friends who
// currently share their location with userID (used for initial map load).
func (s *Service) SnapshotForViewer(ctx context.Context, userID string) ([]Location, error) {
	friendIDs, err := s.friendStore.AcceptedFriendIDsSharingWith(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(friendIDs) == 0 {
		return nil, nil
	}

	// Exclude any friend who has paused sharing globally.
	var visible []string
	for _, id := range friendIDs {
		settings, err := s.store.GetSettings(ctx, id)
		if err != nil {
			return nil, err
		}
		if !settings.SharingPaused {
			visible = append(visible, id)
		}
	}
	return s.store.GetForUsers(ctx, visible)
}

func (s *Service) SetPaused(ctx context.Context, userID string, paused bool) error {
	return s.store.SetPaused(ctx, userID, paused)
}

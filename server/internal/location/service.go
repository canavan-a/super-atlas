package location

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/friends"
)

type Service struct {
	store       *Store
	friendStore *friends.Store
	hub         *Hub
	// profiles caches each user's name and picture id for live pushes.
	// Names can't be changed after sign-up; a new picture calls
	// ForgetProfile.
	profiles sync.Map
}

func NewService(store *Store, friendStore *friends.Store, hub *Hub) *Service {
	return &Service{store: store, friendStore: friendStore, hub: hub}
}

// updatePayload is one friend's position, as pushed over the stream and
// returned by the snapshot endpoint.
type updatePayload struct {
	Type        string   `json:"type"`
	UserID      string   `json:"userId"`
	DisplayName string   `json:"displayName"`
	AvatarID    string   `json:"avatarId,omitempty"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	AccuracyM   *float64 `json:"accuracyM,omitempty"`
	RecordedAt  string   `json:"recordedAt"`
}

func payloadFor(l Location) updatePayload {
	return updatePayload{
		Type:        "location_update",
		UserID:      l.UserID,
		DisplayName: l.DisplayName,
		AvatarID:    l.AvatarID,
		Latitude:    l.Latitude,
		Longitude:   l.Longitude,
		AccuracyM:   l.AccuracyM,
		RecordedAt:  l.RecordedAt.UTC().Format(time.RFC3339),
	}
}

// UpdateLocation stores a new position for userID and, unless the user is in
// ghost mode, pushes it live to every accepted friend with an open
// WebSocket connection.
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64, accuracy *float64, recordedAt time.Time) error {
	if err := s.store.Upsert(ctx, userID, lat, lng, accuracy, recordedAt); err != nil {
		return err
	}

	ghost, err := s.GhostMode(ctx, userID)
	if err != nil || ghost {
		return err
	}

	viewers, err := s.friendStore.AcceptedFriendIDs(ctx, userID)
	if err != nil || len(viewers) == 0 {
		return err
	}

	p, err := s.profile(ctx, userID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(payloadFor(Location{
		UserID:      userID,
		DisplayName: p.name,
		AvatarID:    p.avatarID,
		Latitude:    lat,
		Longitude:   lng,
		AccuracyM:   accuracy,
		RecordedAt:  recordedAt,
	}))
	if err != nil {
		return err
	}

	for _, viewerID := range viewers {
		s.hub.PushTo(viewerID, payload)
	}
	return nil
}

type profile struct{ name, avatarID string }

func (s *Service) profile(ctx context.Context, userID string) (profile, error) {
	if v, ok := s.profiles.Load(userID); ok {
		return v.(profile), nil
	}
	name, avatarID, err := s.store.Profile(ctx, userID)
	if err != nil {
		return profile{}, err
	}
	p := profile{name, avatarID}
	s.profiles.Store(userID, p)
	return p, nil
}

// ForgetProfile drops userID's cached profile, after its picture changes.
func (s *Service) ForgetProfile(userID string) {
	s.profiles.Delete(userID)
}

// SnapshotForViewer returns the last-known positions of userID's friends,
// leaving out anyone in ghost mode (used for initial map load).
func (s *Service) SnapshotForViewer(ctx context.Context, userID string) ([]updatePayload, error) {
	friendIDs, err := s.friendStore.AcceptedFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	locs, err := s.store.GetForUsers(ctx, friendIDs)
	if err != nil {
		return nil, err
	}
	out := make([]updatePayload, 0, len(locs))
	for _, l := range locs {
		out = append(out, payloadFor(l))
	}
	return out, nil
}

func (s *Service) GhostMode(ctx context.Context, userID string) (bool, error) {
	settings, err := s.store.GetSettings(ctx, userID)
	if err != nil {
		return false, err
	}
	return settings.SharingPaused, nil
}

func (s *Service) SetPaused(ctx context.Context, userID string, paused bool) error {
	return s.store.SetPaused(ctx, userID, paused)
}

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
	// ProfileChanged.
	profiles sync.Map
}

func NewService(store *Store, friendStore *friends.Store, hub *Hub) *Service {
	return &Service{store: store, friendStore: friendStore, hub: hub}
}

// updatePayload is one friend's position, as pushed over the stream and
// returned by the snapshot endpoint. SpeedMps is how fast they were going
// (when their phone says), and HereSince when they arrived near this spot.
type updatePayload struct {
	Type        string   `json:"type"`
	UserID      string   `json:"userId"`
	DisplayName string   `json:"displayName"`
	AvatarID    string   `json:"avatarId,omitempty"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	AccuracyM   *float64 `json:"accuracyM,omitempty"`
	SpeedMps    *float64 `json:"speedMps,omitempty"`
	RecordedAt  string   `json:"recordedAt"`
	HereSince   string   `json:"hereSince,omitempty"`
}

func payloadFor(l Location) updatePayload {
	p := updatePayload{
		Type:        "location_update",
		UserID:      l.UserID,
		DisplayName: l.DisplayName,
		AvatarID:    l.AvatarID,
		Latitude:    l.Latitude,
		Longitude:   l.Longitude,
		AccuracyM:   l.AccuracyM,
		SpeedMps:    l.SpeedMps,
		RecordedAt:  l.RecordedAt.UTC().Format(time.RFC3339),
	}
	if !l.HereSince.IsZero() {
		p.HereSince = l.HereSince.UTC().Format(time.RFC3339)
	}
	return p
}

// UpdateLocation stores a new position for userID, pushes it to userID's own
// open connections as a self_update and, unless the user is in ghost mode,
// live to every accepted friend with one. A fix older than the stored one is
// dropped.
func (s *Service) UpdateLocation(ctx context.Context, userID string, fix Fix) error {
	hereSince, ok, err := s.store.Upsert(ctx, userID, fix)
	if err != nil || !ok {
		return err
	}

	p, err := s.profile(ctx, userID)
	if err != nil {
		return err
	}
	update := payloadFor(Location{
		UserID:      userID,
		DisplayName: p.name,
		AvatarID:    p.avatarID,
		Latitude:    fix.Latitude,
		Longitude:   fix.Longitude,
		AccuracyM:   fix.AccuracyM,
		SpeedMps:    fix.SpeedMps,
		RecordedAt:  fix.RecordedAt,
		HereSince:   hereSince,
	})

	// Your own open maps get your position too, ghost mode or not: the web
	// build has no location of its own and shows your phone's.
	update.Type = "self_update"
	if own, err := json.Marshal(update); err == nil {
		s.hub.PushTo(userID, own)
	}

	ghost, err := s.GhostMode(ctx, userID)
	if err != nil || ghost {
		return err
	}

	viewers, err := s.friendStore.AcceptedFriendIDs(ctx, userID)
	if err != nil || len(viewers) == 0 {
		return err
	}

	update.Type = "location_update"
	payload, err := json.Marshal(update)
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

// ProfileChanged drops userID's cached profile after its picture changes,
// and re-sends its last position (Refresh) so friends with the map open see
// the new picture without waiting for userID to move.
func (s *Service) ProfileChanged(ctx context.Context, userID string) {
	s.profiles.Delete(userID)
	s.Refresh(ctx, userID)
}

// removedPayload tells a viewer to take userID off their map.
type removedPayload struct {
	Type   string `json:"type"`
	UserID string `json:"userId"`
}

// Refresh brings userID up to date on the open maps of all their friends,
// after their ghost mode or picture changed: their last position, or, in
// ghost mode (or with none yet), word to drop them.
func (s *Service) Refresh(ctx context.Context, userID string) {
	friendIDs, err := s.friendStore.AcceptedFriendIDs(ctx, userID)
	if err != nil || len(friendIDs) == 0 {
		return
	}
	locs, err := s.store.GetForUsers(ctx, []string{userID})
	if err != nil {
		return
	}
	payload, err := json.Marshal(removedPayload{Type: "location_removed", UserID: userID})
	if len(locs) > 0 {
		payload, err = json.Marshal(payloadFor(locs[0]))
	}
	if err != nil {
		return
	}
	for _, id := range friendIDs {
		s.hub.PushTo(id, payload)
	}
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

// Own returns userID's own last position for their own map (ghost mode or
// not), or nil when they have none yet.
func (s *Service) Own(ctx context.Context, userID string) (*updatePayload, error) {
	l, err := s.store.GetOwn(ctx, userID)
	if err != nil || l == nil {
		return nil, err
	}
	p := payloadFor(*l)
	p.Type = "self_update"
	return &p, nil
}

func (s *Service) GhostMode(ctx context.Context, userID string) (bool, error) {
	settings, err := s.store.GetSettings(ctx, userID)
	if err != nil {
		return false, err
	}
	return settings.SharingPaused, nil
}

// SetPaused turns ghost mode on or off, and takes userID off (or puts them
// back on) their friends' open maps right away.
func (s *Service) SetPaused(ctx context.Context, userID string, paused bool) error {
	if err := s.store.SetPaused(ctx, userID, paused); err != nil {
		return err
	}
	s.Refresh(ctx, userID)
	return nil
}

package npc

import (
	"context"
	"embed"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/avatar"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/google/uuid"
)

const (
	// A driving NPC sends its position this often: the app slides a dot to
	// each new position over a second, so the car moves without stopping.
	driveEvery = time.Second
	// A parked one sends it this often, so a new friend sees them soon and
	// they never look out of date.
	parkedEvery = 10 * time.Second
)

// avatars holds each NPC's profile picture, named after their email before
// the @ (see avatars/README.md).
//
//go:embed avatars/*.png
var avatars embed.FS

// Service keeps the NPCs' accounts and streams their positions.
type Service struct {
	authStore       *auth.Store
	friendStore     *friends.Store
	locationService *location.Service
	avatarStore     *avatar.Store
	// userIDs maps each NPC's user id to its index in cast. Set by Setup,
	// read-only after.
	userIDs map[string]int
}

func NewService(authStore *auth.Store, friendStore *friends.Store, locationService *location.Service, avatarStore *avatar.Store) *Service {
	return &Service{authStore: authStore, friendStore: friendStore, locationService: locationService, avatarStore: avatarStore}
}

// Setup makes sure every NPC has an account and a profile picture, creating
// any that are missing.
func (s *Service) Setup(ctx context.Context) error {
	s.userIDs = make(map[string]int, len(cast))
	for i, n := range cast {
		u, err := s.authStore.GetUserByEmail(ctx, n.Email)
		var id string
		switch {
		case errors.Is(err, auth.ErrNotFound):
			id = uuid.NewString()
			if err := s.authStore.CreateUser(ctx, id, n.Email, n.Name); err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			id = u.ID
		}
		s.userIDs[id] = i
		if err := s.setAvatar(ctx, id, n); err != nil {
			return err
		}
	}
	return nil
}

// setAvatar gives n their picture unless they already have one, so the
// picture's id (and every app's cached copy) stays the same across restarts.
func (s *Service) setAvatar(ctx context.Context, userID string, n NPC) error {
	if has, err := s.avatarStore.Has(ctx, userID); err != nil || has {
		return err
	}
	data, err := avatars.ReadFile("avatars/" + strings.Split(n.Email, "@")[0] + ".png")
	if err != nil {
		return err
	}
	jpeg, err := avatar.Process(data)
	if err != nil {
		return err
	}
	_, err = s.avatarStore.Set(ctx, userID, jpeg)
	return err
}

// IsNPC says whether userID is one of the NPCs, whose friend requests are
// accepted at once (friends.Service.SetAutoAccept).
func (s *Service) IsNPC(userID string) bool {
	_, ok := s.userIDs[userID]
	return ok
}

// Run streams each NPC's position, to its friends only, until ctx ends. An
// NPC nobody has friended sends nothing.
func (s *Service) Run(ctx context.Context) {
	lastSent := make(map[string]time.Time, len(s.userIDs))
	ticker := time.NewTicker(driveEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for id, i := range s.userIDs {
				lngLat, driving := position(cast[i], now)
				if !driving && now.Sub(lastSent[id]) < parkedEvery {
					continue
				}
				viewers, err := s.friendStore.AcceptedFriendIDs(ctx, id)
				if err != nil || len(viewers) == 0 {
					continue
				}
				speed := speedAt(cast[i], now)
				fix := location.Fix{Latitude: lngLat[1], Longitude: lngLat[0], SpeedMps: &speed, RecordedAt: now}
				if err := s.locationService.UpdateLocation(ctx, id, fix); err != nil {
					log.Printf("npc %s: %v", cast[i].Email, err)
					continue
				}
				lastSent[id] = now
			}
		}
	}
}

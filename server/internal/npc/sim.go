package npc

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
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

// Service keeps the NPCs' accounts and streams their positions.
type Service struct {
	authStore       *auth.Store
	friendStore     *friends.Store
	locationService *location.Service
	// userIDs maps each NPC's user id to its index in cast. Set by Setup,
	// read-only after.
	userIDs map[string]int
}

func NewService(authStore *auth.Store, friendStore *friends.Store, locationService *location.Service) *Service {
	return &Service{authStore: authStore, friendStore: friendStore, locationService: locationService}
}

// Setup makes sure every NPC has an account, creating any that are missing.
func (s *Service) Setup(ctx context.Context) error {
	s.userIDs = make(map[string]int, len(cast))
	for i, n := range cast {
		u, err := s.authStore.GetUserByEmail(ctx, n.Email)
		if errors.Is(err, auth.ErrNotFound) {
			id := uuid.NewString()
			if err := s.authStore.CreateUser(ctx, id, n.Email, n.Name); err != nil {
				return err
			}
			s.userIDs[id] = i
			continue
		}
		if err != nil {
			return err
		}
		s.userIDs[u.ID] = i
	}
	return nil
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
				if err := s.locationService.UpdateLocation(ctx, id, lngLat[1], lngLat[0], nil, now); err != nil {
					log.Printf("npc %s: %v", cast[i].Email, err)
					continue
				}
				lastSent[id] = now
			}
		}
	}
}

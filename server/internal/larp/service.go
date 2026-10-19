package larp

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/google/uuid"
)

var fakeNames = []string{
	"Riley Fox", "Jordan Vale", "Casey Rivers", "Morgan Blake",
	"Avery Stone", "Quinn Harper", "Skyler Reed", "Rowan Pike",
}

const (
	fakeFriendCount = 6
	// Frequent, small steps read as continuous movement on the client instead
	// of jumps; the client also re-renders markers instantly on each update
	// with no interpolation, so smoothness has to come from the tick rate.
	simTickInterval = 1500 * time.Millisecond
	simStepDegrees  = 0.0003
)

// defaultBase is used as the center point for simulated friends if the
// owner has no known location yet (matches the app's initial map camera).
var defaultBase = struct{ Lat, Lng float64 }{Lat: 40.7, Lng: -74.0}

type Service struct {
	store           *Store
	authStore       *auth.Store
	friendStore     *friends.Store
	locationStore   *location.Store
	locationService *location.Service

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewService(store *Store, authStore *auth.Store, friendStore *friends.Store, locationStore *location.Store, locationService *location.Service) *Service {
	return &Service{
		store:           store,
		authStore:       authStore,
		friendStore:     friendStore,
		locationStore:   locationStore,
		locationService: locationService,
		cancels:         make(map[string]context.CancelFunc),
	}
}

// Toggle turns simulated fake friends on (creating them, friending the
// owner, and starting a background simulation) or off (deleting them and
// stopping the simulation) for ownerID. Returns the resulting active state.
func (s *Service) Toggle(ctx context.Context, ownerID string) (bool, error) {
	active, err := s.store.IsActive(ctx, ownerID)
	if err != nil {
		return false, err
	}
	if active {
		s.stopSimulation(ownerID)
		if err := s.store.DeleteForOwner(ctx, ownerID); err != nil {
			return false, err
		}
		return false, nil
	}

	base := s.baseFor(ctx, ownerID)
	fakeIDs, err := s.createFakeFriends(ctx, ownerID, base)
	if err != nil {
		return false, err
	}
	s.startSimulation(ownerID, fakeIDs, base)
	return true, nil
}

func (s *Service) baseFor(ctx context.Context, ownerID string) struct{ Lat, Lng float64 } {
	locs, err := s.locationStore.GetForUsers(ctx, []string{ownerID})
	if err != nil || len(locs) == 0 {
		return defaultBase
	}
	return struct{ Lat, Lng float64 }{Lat: locs[0].Latitude, Lng: locs[0].Longitude}
}

func (s *Service) createFakeFriends(ctx context.Context, ownerID string, base struct{ Lat, Lng float64 }) ([]string, error) {
	fakeIDs := make([]string, 0, fakeFriendCount)
	for i := 0; i < fakeFriendCount; i++ {
		id := uuid.NewString()
		email := fmt.Sprintf("larp%d-%s@fake.local", i, id[:8])
		displayName := fakeNames[i%len(fakeNames)]
		if err := s.authStore.CreateUser(ctx, id, email, displayName); err != nil {
			return nil, err
		}
		if err := s.store.RecordFakeUser(ctx, id, ownerID); err != nil {
			return nil, err
		}

		friendshipID := uuid.NewString()
		if err := s.friendStore.Create(ctx, friendshipID, ownerID, id); err != nil {
			return nil, err
		}
		if err := s.friendStore.Accept(ctx, friendshipID); err != nil {
			return nil, err
		}

		lat, lng := jitter(base)
		if err := s.locationService.UpdateLocation(ctx, id, lat, lng, nil, time.Now()); err != nil {
			return nil, err
		}
		fakeIDs = append(fakeIDs, id)
	}
	return fakeIDs, nil
}

func jitter(base struct{ Lat, Lng float64 }) (float64, float64) {
	lat := base.Lat + (rand.Float64()*2-1)*0.01
	lng := base.Lng + (rand.Float64()*2-1)*0.01
	return lat, lng
}

func (s *Service) startSimulation(ownerID string, fakeIDs []string, base struct{ Lat, Lng float64 }) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[ownerID] = cancel
	s.mu.Unlock()

	positions := make(map[string][2]float64, len(fakeIDs))
	for _, id := range fakeIDs {
		lat, lng := jitter(base)
		positions[id] = [2]float64{lat, lng}
	}

	go func() {
		ticker := time.NewTicker(simTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, id := range fakeIDs {
					pos := positions[id]
					pos[0] += (rand.Float64()*2 - 1) * simStepDegrees
					pos[1] += (rand.Float64()*2 - 1) * simStepDegrees
					positions[id] = pos
					_ = s.locationService.UpdateLocation(ctx, id, pos[0], pos[1], nil, time.Now())
				}
			}
		}
	}()
}

func (s *Service) stopSimulation(ownerID string) {
	s.mu.Lock()
	cancel, ok := s.cancels[ownerID]
	delete(s.cancels, ownerID)
	s.mu.Unlock()
	if ok {
		cancel()
	}
}

package friends

import (
	"context"
	"errors"
	"strings"

	"github.com/canavan-a/open-location/server/internal/auth"
	"github.com/google/uuid"
)

var (
	ErrSelfRequest       = errors.New("cannot friend yourself")
	ErrAlreadyExists     = errors.New("friendship already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrNotYourRequest    = errors.New("not your request to act on")
	ErrNotPending        = errors.New("friendship is not pending")
)

type Service struct {
	store     *Store
	authStore *auth.Store
}

func NewService(store *Store, authStore *auth.Store) *Service {
	return &Service{store: store, authStore: authStore}
}

// SendRequest creates a pending friendship from requesterID to the user with addresseeEmail.
func (s *Service) SendRequest(ctx context.Context, requesterID, addresseeEmail string) (*Friendship, error) {
	addressee, err := s.authStore.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(addresseeEmail)))
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if addressee.ID == requesterID {
		return nil, ErrSelfRequest
	}

	if _, err := s.store.GetByUsers(ctx, requesterID, addressee.ID); err == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	id := uuid.NewString()
	if err := s.store.Create(ctx, id, requesterID, addressee.ID); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Accept(ctx context.Context, friendshipID, userID string) error {
	f, err := s.store.GetByID(ctx, friendshipID)
	if err != nil {
		return err
	}
	if f.AddresseeID != userID {
		return ErrNotYourRequest
	}
	if f.Status != "pending" {
		return ErrNotPending
	}
	return s.store.Accept(ctx, friendshipID)
}

// Decline handles both rejecting a pending request and removing an accepted friendship.
func (s *Service) Remove(ctx context.Context, friendshipID, userID string) error {
	f, err := s.store.GetByID(ctx, friendshipID)
	if err != nil {
		return err
	}
	if f.RequesterID != userID && f.AddresseeID != userID {
		return ErrNotYourRequest
	}
	return s.store.Delete(ctx, friendshipID)
}

func (s *Service) SetSharing(ctx context.Context, friendshipID, userID string, sharing bool) error {
	return s.store.SetSharing(ctx, friendshipID, userID, sharing)
}

func (s *Service) List(ctx context.Context, userID string) ([]FriendView, error) {
	return s.store.ListForUser(ctx, userID)
}

// GetDisplayName returns otherID's display name, but only if userID is
// otherID (looking up their own name) or the two are accepted friends.
func (s *Service) GetDisplayName(ctx context.Context, userID, otherID string) (string, error) {
	if userID == otherID {
		u, err := s.authStore.GetUserByID(ctx, otherID)
		if err != nil {
			if errors.Is(err, auth.ErrNotFound) {
				return "", ErrNotFound
			}
			return "", err
		}
		return u.DisplayName, nil
	}
	return s.store.AcceptedFriendDisplayName(ctx, userID, otherID)
}

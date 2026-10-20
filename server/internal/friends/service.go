package friends

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/google/uuid"
)

var (
	ErrSelfRequest    = errors.New("cannot friend yourself")
	ErrAlreadyExists  = errors.New("friendship already exists")
	ErrUserNotFound   = errors.New("user not found")
	ErrNotYourRequest = errors.New("not your request to act on")
	ErrNotPending     = errors.New("friendship is not pending")
	ErrInvalidInvite  = errors.New("invite link is invalid")
)

// Notifier tells someone about a friend event (implemented by the notify
// package; declared here so friends doesn't depend on it).
type Notifier interface {
	Notify(userID, title, body string)
}

type Service struct {
	store     *Store
	authStore *auth.Store
	// notifier is nil when nothing should be told (tests, larp).
	notifier Notifier
}

func NewService(store *Store, authStore *auth.Store, notifier Notifier) *Service {
	return &Service{store: store, authStore: authStore, notifier: notifier}
}

// notify sends to toID about fromID; failures to look up the name are
// ignored, since a notification is never worth failing the request over.
func (s *Service) notify(toID, fromID string, pending bool) {
	if s.notifier == nil {
		return
	}
	from, err := s.authStore.GetUserByID(context.Background(), fromID)
	if err != nil {
		return
	}
	if pending {
		s.notifier.Notify(toID, from.DisplayName+" wants to be friends", "Open Super Atlas to accept.")
	} else {
		s.notifier.Notify(toID, from.DisplayName+" is now your friend", "You can see each other on the map.")
	}
}

// SendRequest creates a pending friendship from requesterID to the user with
// addresseeEmail. If that user already sent requesterID a request, this
// accepts it instead.
func (s *Service) SendRequest(ctx context.Context, requesterID, addresseeEmail string) (*Friendship, error) {
	addressee, err := s.authStore.GetUserByEmail(ctx, auth.NormalizeEmail(addresseeEmail))
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if addressee.ID == requesterID {
		return nil, ErrSelfRequest
	}

	existing, err := s.store.GetByUsers(ctx, requesterID, addressee.ID)
	switch {
	case err == nil:
		if existing.Status == "pending" && existing.AddresseeID == requesterID {
			if err := s.store.Accept(ctx, existing.ID); err != nil {
				return nil, err
			}
			s.notify(addressee.ID, requesterID, false)
			return s.store.GetByID(ctx, existing.ID)
		}
		return nil, ErrAlreadyExists
	case !errors.Is(err, ErrNotFound):
		return nil, err
	}

	id := uuid.NewString()
	if err := s.store.Create(ctx, id, requesterID, addressee.ID); err != nil {
		return nil, err
	}
	s.notify(addressee.ID, requesterID, true)
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
	if err := s.store.Accept(ctx, friendshipID); err != nil {
		return err
	}
	s.notify(f.RequesterID, userID, false)
	return nil
}

// Remove handles declining or cancelling a pending request and removing an
// accepted friendship.
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

func (s *Service) List(ctx context.Context, userID string) ([]FriendView, error) {
	return s.store.ListForUser(ctx, userID)
}

// InviteCode returns userID's personal invite code, creating it on first use.
func (s *Service) InviteCode(ctx context.Context, userID string) (string, error) {
	code, err := s.store.InviteCode(ctx, userID)
	if err != nil || code != "" {
		return code, err
	}
	return s.ResetInviteCode(ctx, userID)
}

// ResetInviteCode replaces userID's invite code, so links shared before stop
// working.
func (s *Service) ResetInviteCode(ctx context.Context, userID string) (string, error) {
	code, err := newInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.store.SetInviteCode(ctx, userID, code); err != nil {
		return "", err
	}
	return code, nil
}

// InviteOwner returns the display name of the person an invite code belongs to.
func (s *Service) InviteOwner(ctx context.Context, code string) (string, error) {
	_, name, err := s.store.UserByInviteCode(ctx, code)
	if errors.Is(err, ErrNotFound) {
		return "", ErrInvalidInvite
	}
	return name, err
}

// AcceptInvite makes userID and the code's owner friends straight away. It's
// a no-op if they already are, and accepts a pending request either way.
func (s *Service) AcceptInvite(ctx context.Context, userID, code string) (string, error) {
	ownerID, ownerName, err := s.store.UserByInviteCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidInvite
		}
		return "", err
	}
	if ownerID == userID {
		return "", ErrSelfRequest
	}

	existing, err := s.store.GetByUsers(ctx, userID, ownerID)
	switch {
	case err == nil:
		if existing.Status == "pending" {
			if err := s.store.Accept(ctx, existing.ID); err != nil {
				return "", err
			}
			s.notify(ownerID, userID, false)
		}
		return ownerName, nil
	case !errors.Is(err, ErrNotFound):
		return "", err
	}

	id := uuid.NewString()
	if err := s.store.Create(ctx, id, ownerID, userID); err != nil {
		return "", err
	}
	if err := s.store.Accept(ctx, id); err != nil {
		return "", err
	}
	s.notify(ownerID, userID, false)
	return ownerName, nil
}

// newInviteCode returns 60 random bits as 12 lowercase base32 characters:
// short enough to read aloud, long enough that guessing one (behind a rate
// limit) isn't practical.
func newInviteCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return strings.ToLower(enc[:12]), nil
}

package friends

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

const timeLayout = "2006-01-02T15:04:05.999Z07:00"

type Friendship struct {
	ID                string
	RequesterID       string
	AddresseeID       string
	Status            string // "pending" | "accepted"
	RequesterSharing  bool
	AddresseeSharing  bool
	CreatedAt         time.Time
}

// FriendView is a friendship joined with the *other* user's public info,
// from the perspective of the caller.
type FriendView struct {
	FriendshipID string
	FriendID     string
	Email        string
	DisplayName  string
	Status       string
	// Sharing is true if the caller has sharing enabled toward this friend.
	Sharing bool
	// IsIncomingRequest is true if this pending friendship was sent *to* the caller.
	IsIncomingRequest bool
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, id, requesterID, addresseeID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO friendships (id, requester_id, addressee_id) VALUES (?, ?, ?)`,
		id, requesterID, addresseeID,
	)
	return err
}

func (s *Store) GetByUsers(ctx context.Context, userA, userB string) (*Friendship, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, requester_id, addressee_id, status, requester_sharing, addressee_sharing, created_at
		 FROM friendships
		 WHERE (requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)`,
		userA, userB, userB, userA,
	)
	return scanFriendship(row)
}

func (s *Store) GetByID(ctx context.Context, id string) (*Friendship, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, requester_id, addressee_id, status, requester_sharing, addressee_sharing, created_at
		 FROM friendships WHERE id = ?`, id)
	return scanFriendship(row)
}

func scanFriendship(row *sql.Row) (*Friendship, error) {
	var f Friendship
	var requesterSharing, addresseeSharing int
	var createdAt string
	if err := row.Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &requesterSharing, &addresseeSharing, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	f.RequesterSharing = requesterSharing == 1
	f.AddresseeSharing = addresseeSharing == 1
	f.CreatedAt, _ = time.Parse(timeLayout, createdAt)
	return &f, nil
}

func (s *Store) Accept(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE friendships SET status = 'accepted', updated_at = ? WHERE id = ? AND status = 'pending'`,
		time.Now().UTC().Format(timeLayout), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM friendships WHERE id = ?`, id)
	return err
}

// SetSharing sets the sharing flag for whichever side of the friendship userID is on.
func (s *Store) SetSharing(ctx context.Context, id, userID string, sharing bool) error {
	f, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	col := "addressee_sharing"
	if f.RequesterID == userID {
		col = "requester_sharing"
	} else if f.AddresseeID != userID {
		return ErrNotFound
	}
	val := 0
	if sharing {
		val = 1
	}
	_, err = s.db.ExecContext(ctx, `UPDATE friendships SET `+col+` = ?, updated_at = ? WHERE id = ?`,
		val, time.Now().UTC().Format(timeLayout), id)
	return err
}

// ListForUser returns every friendship (pending and accepted) involving userID,
// joined with the other party's public info.
func (s *Store) ListForUser(ctx context.Context, userID string) ([]FriendView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.requester_id, f.addressee_id, f.status, f.requester_sharing, f.addressee_sharing,
		       u.id, u.email, u.display_name
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.addressee_id ELSE f.requester_id END
		WHERE f.requester_id = ? OR f.addressee_id = ?
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []FriendView
	for rows.Next() {
		var requesterID, addresseeID string
		var requesterSharing, addresseeSharing int
		var v FriendView
		if err := rows.Scan(&v.FriendshipID, &requesterID, &addresseeID, &v.Status, &requesterSharing, &addresseeSharing,
			&v.FriendID, &v.Email, &v.DisplayName); err != nil {
			return nil, err
		}
		if requesterID == userID {
			v.Sharing = requesterSharing == 1
		} else {
			v.Sharing = addresseeSharing == 1
			v.IsIncomingRequest = v.Status == "pending"
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// AcceptedFriendDisplayName returns the display name of otherID, but only if
// userID and otherID have an accepted friendship.
func (s *Store) AcceptedFriendDisplayName(ctx context.Context, userID, otherID string) (string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.display_name
		FROM friendships f
		JOIN users u ON u.id = ?
		WHERE f.status = 'accepted'
		  AND ((f.requester_id = ? AND f.addressee_id = ?) OR (f.requester_id = ? AND f.addressee_id = ?))
	`, otherID, userID, otherID, otherID, userID)

	var displayName string
	if err := row.Scan(&displayName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return displayName, nil
}

// AcceptedFriendIDs returns the IDs of userID's accepted friends who currently
// have sharing enabled toward userID.
func (s *Store) AcceptedFriendIDsSharingWith(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN requester_id = ? THEN addressee_id ELSE requester_id END
		FROM friendships
		WHERE status = 'accepted'
		  AND ((requester_id = ? AND addressee_sharing = 1) OR (addressee_id = ? AND requester_sharing = 1))
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

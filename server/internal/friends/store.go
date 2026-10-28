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
	ID          string
	RequesterID string
	AddresseeID string
	Status      string // "pending" | "accepted"
	CreatedAt   time.Time
}

// FriendView is a friendship joined with the *other* user's public info,
// from the perspective of the caller.
type FriendView struct {
	FriendshipID string `json:"friendshipId"`
	FriendID     string `json:"friendId"`
	DisplayName  string `json:"displayName"`
	AvatarID     string `json:"avatarId,omitempty"`
	Status       string `json:"status"`
	// IsIncomingRequest is true if this pending friendship was sent *to* the caller.
	IsIncomingRequest bool `json:"isIncomingRequest"`
	// LastSeenAt is when the friend's location last updated; only set for
	// accepted friends who aren't in ghost mode.
	LastSeenAt *string `json:"lastSeenAt,omitempty"`
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
		`SELECT id, requester_id, addressee_id, status, created_at
		 FROM friendships
		 WHERE (requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)`,
		userA, userB, userB, userA,
	)
	return scanFriendship(row)
}

func (s *Store) GetByID(ctx context.Context, id string) (*Friendship, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, requester_id, addressee_id, status, created_at FROM friendships WHERE id = ?`, id)
	return scanFriendship(row)
}

func scanFriendship(row *sql.Row) (*Friendship, error) {
	var f Friendship
	var createdAt string
	if err := row.Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
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

// ListForUser returns every friendship (pending and accepted) involving userID,
// joined with the other party's public info.
func (s *Store) ListForUser(ctx context.Context, userID string) ([]FriendView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.addressee_id, f.status, u.id, u.display_name, COALESCE(a.id, ''),
		       CASE WHEN f.status = 'accepted' AND COALESCE(us.sharing_paused, 0) = 0 THEN l.recorded_at END
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.addressee_id ELSE f.requester_id END
		LEFT JOIN user_settings us ON us.user_id = u.id
		LEFT JOIN locations l ON l.user_id = u.id
		LEFT JOIN avatars a ON a.user_id = u.id
		WHERE f.requester_id = ? OR f.addressee_id = ?
		ORDER BY u.display_name COLLATE NOCASE
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := []FriendView{}
	for rows.Next() {
		var addresseeID string
		var lastSeen sql.NullString
		var v FriendView
		if err := rows.Scan(&v.FriendshipID, &addresseeID, &v.Status, &v.FriendID, &v.DisplayName, &v.AvatarID, &lastSeen); err != nil {
			return nil, err
		}
		v.IsIncomingRequest = v.Status == "pending" && addresseeID == userID
		if lastSeen.Valid {
			v.LastSeenAt = &lastSeen.String
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// AcceptedFriendIDs returns the IDs of userID's accepted friends.
func (s *Store) AcceptedFriendIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN requester_id = ? THEN addressee_id ELSE requester_id END
		FROM friendships
		WHERE status = 'accepted' AND (requester_id = ? OR addressee_id = ?)
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

// InviteCode returns userID's invite code, or "" if they don't have one yet.
func (s *Store) InviteCode(ctx context.Context, userID string) (string, error) {
	var code sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT invite_code FROM users WHERE id = ?`, userID).Scan(&code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return code.String, nil
}

func (s *Store) SetInviteCode(ctx context.Context, userID, code string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET invite_code = ? WHERE id = ?`, code, userID)
	return err
}

// UserByInviteCode returns the ID and display name of the code's owner.
func (s *Store) UserByInviteCode(ctx context.Context, code string) (id, displayName string, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT id, display_name FROM users WHERE invite_code = ?`, code).Scan(&id, &displayName)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return id, displayName, err
}

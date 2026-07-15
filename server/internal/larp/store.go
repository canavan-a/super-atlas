package larp

import (
	"context"
	"database/sql"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) RecordFakeUser(ctx context.Context, fakeUserID, ownerID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO larp_fake_users (fake_user_id, owner_id) VALUES (?, ?)`,
		fakeUserID, ownerID,
	)
	return err
}

func (s *Store) IsActive(ctx context.Context, ownerID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM larp_fake_users WHERE owner_id = ?`, ownerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) FakeUserIDsForOwner(ctx context.Context, ownerID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT fake_user_id FROM larp_fake_users WHERE owner_id = ?`, ownerID)
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

// DeleteForOwner deletes the fake users themselves; cascading foreign keys
// clean up their friendships, locations, settings, and larp_fake_users rows.
func (s *Store) DeleteForOwner(ctx context.Context, ownerID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM users WHERE id IN (SELECT fake_user_id FROM larp_fake_users WHERE owner_id = ?)
	`, ownerID)
	return err
}

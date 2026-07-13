package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	CreatedAt    time.Time
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const timeLayout = "2006-01-02T15:04:05.999Z07:00"

func (s *Store) CreateUser(ctx context.Context, id, email, passwordHash, displayName string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, display_name) VALUES (?, ?, ?, ?)`,
		id, email, passwordHash, displayName,
	)
	return err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, created_at FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, created_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(timeLayout, createdAt)
	return &u, nil
}

func (s *Store) CreateRefreshToken(ctx context.Context, id, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, tokenHash, expiresAt.UTC().Format(timeLayout),
	)
	return err
}

func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = ?`, tokenHash)
	var rt RefreshToken
	var expiresAt string
	var revokedAt sql.NullString
	if err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rt.ExpiresAt, _ = time.Parse(timeLayout, expiresAt)
	if revokedAt.Valid {
		t, _ := time.Parse(timeLayout, revokedAt.String)
		rt.RevokedAt = &t
	}
	return &rt, nil
}

func (s *Store) RevokeRefreshToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE id = ?`,
		time.Now().UTC().Format(timeLayout), id,
	)
	return err
}

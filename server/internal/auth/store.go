package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
}

type LoginCode struct {
	ID        string
	Email     string
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	UsedAt    *time.Time
}

type Session struct {
	ID         string
	UserID     string
	LastUsedAt time.Time
	ExpiresAt  time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const timeLayout = "2006-01-02T15:04:05.999Z07:00"

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

func (s *Store) CreateUser(ctx context.Context, id, email, displayName string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, display_name) VALUES (?, ?, ?)`,
		id, email, displayName,
	)
	return err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, created_at FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, created_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.CreatedAt = parseTime(createdAt)
	return &u, nil
}

func (s *Store) CreateLoginCode(ctx context.Context, id, email, codeHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO login_codes (id, email, code_hash, expires_at) VALUES (?, ?, ?, ?)`,
		id, email, codeHash, formatTime(expiresAt),
	)
	return err
}

// LatestLoginCode returns the most recently issued code for email. Only the
// latest one counts: requesting a new code retires the previous one.
func (s *Store) LatestLoginCode(ctx context.Context, email string) (*LoginCode, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, code_hash, expires_at, attempts, used_at FROM login_codes
		 WHERE email = ? ORDER BY rowid DESC LIMIT 1`, email)
	var c LoginCode
	var expiresAt string
	var usedAt sql.NullString
	if err := row.Scan(&c.ID, &c.Email, &c.CodeHash, &expiresAt, &c.Attempts, &usedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.ExpiresAt = parseTime(expiresAt)
	if usedAt.Valid {
		t := parseTime(usedAt.String)
		c.UsedAt = &t
	}
	return &c, nil
}

func (s *Store) IncrementLoginCodeAttempts(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE login_codes SET attempts = attempts + 1 WHERE id = ?`, id)
	return err
}

func (s *Store) MarkLoginCodeUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE login_codes SET used_at = ? WHERE id = ?`, formatTime(time.Now()), id)
	return err
}

// DeleteExpiredLoginCodes keeps the table from growing without bound.
func (s *Store) DeleteExpiredLoginCodes(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM login_codes WHERE expires_at < ?`, formatTime(now))
	return err
}

func (s *Store) CreateSession(ctx context.Context, id, userID, tokenHash string, now, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, last_used_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, formatTime(now), formatTime(expiresAt),
	)
	return err
}

func (s *Store) GetSessionByHash(ctx context.Context, tokenHash string) (*Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, last_used_at, expires_at FROM sessions WHERE token_hash = ?`, tokenHash)
	var sess Session
	var lastUsedAt, expiresAt string
	if err := row.Scan(&sess.ID, &sess.UserID, &lastUsedAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	sess.LastUsedAt = parseTime(lastUsedAt)
	sess.ExpiresAt = parseTime(expiresAt)
	return &sess, nil
}

func (s *Store) TouchSession(ctx context.Context, id string, now, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET last_used_at = ?, expires_at = ? WHERE id = ?`,
		formatTime(now), formatTime(expiresAt), id,
	)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, formatTime(now))
	return err
}

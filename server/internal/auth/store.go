package auth

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID          string
	Email       string
	DisplayName string
	// AvatarID is the profile picture's id (avatar package), "" for none.
	AvatarID  string
	CreatedAt time.Time
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
		userSelect+` WHERE u.email = ?`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		userSelect+` WHERE u.id = ?`, id)
	return scanUser(row)
}

const userSelect = `SELECT u.id, u.email, u.display_name, COALESCE(a.id, ''), u.created_at
	FROM users u LEFT JOIN avatars a ON a.user_id = u.id`

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var createdAt string
	if err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.AvatarID, &createdAt); err != nil {
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

// DeleteUser deletes the account and everything tied to it. Deleting the
// users row cascades to its sessions, push tokens, friendships, location,
// settings and profile picture; its pending sign-in codes and any dev-mode
// "larp" fake friends it owns go here too. That frees its place under the
// user cap.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var email string
	if err := tx.QueryRowContext(ctx, `SELECT lower(email) FROM users WHERE id = ?`, id).Scan(&email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	for _, q := range []string{
		`DELETE FROM users WHERE id IN (SELECT fake_user_id FROM larp_fake_users WHERE owner_id = ?)`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM login_codes WHERE lower(email) = ?`, email); err != nil {
		return err
	}
	return tx.Commit()
}

// countedUsers leaves out the accounts that aren't people: the built-in
// NPCs (@lunargen.com) and dev-mode "larp" fake friends.
const countedUsers = `
	FROM users
	WHERE email NOT LIKE '%@lunargen.com'
	  AND id NOT IN (SELECT fake_user_id FROM larp_fake_users)`

// CountUsers is how many real people have an account: what the user cap
// limits.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)`+countedUsers).Scan(&n)
	return n, err
}

// UserCap is the most accounts there may be; nobody new can sign up past it.
func (s *Store) UserCap(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT CAST(value AS INTEGER) FROM app_settings WHERE key = 'user_cap'`).Scan(&n)
	return n, err
}

func (s *Store) SetUserCap(ctx context.Context, n int) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO app_settings (key, value) VALUES ('user_cap', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, strconv.Itoa(n))
	return err
}

// Member is one counted account, as superatlas-admin lists them.
type Member struct {
	Email       string
	DisplayName string
	CreatedAt   time.Time
}

// ListUsers lists the counted accounts, oldest first.
func (s *Store) ListUsers(ctx context.Context) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT email, display_name, created_at`+countedUsers+` ORDER BY created_at, email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		var created string
		if err := rows.Scan(&m.Email, &m.DisplayName, &created); err != nil {
			return nil, err
		}
		m.CreatedAt = parseTime(created)
		out = append(out, m)
	}
	return out, rows.Err()
}

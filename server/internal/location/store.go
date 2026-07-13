package location

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

const timeLayout = "2006-01-02T15:04:05.999Z07:00"

type Location struct {
	UserID     string
	Latitude   float64
	Longitude  float64
	AccuracyM  *float64
	RecordedAt time.Time
	UpdatedAt  time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Upsert(ctx context.Context, userID string, lat, lng float64, accuracy *float64, recordedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO locations (user_id, latitude, longitude, accuracy_m, recorded_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			accuracy_m = excluded.accuracy_m,
			recorded_at = excluded.recorded_at,
			updated_at = excluded.updated_at
		WHERE excluded.recorded_at >= locations.recorded_at
	`, userID, lat, lng, accuracy, recordedAt.UTC().Format(timeLayout), time.Now().UTC().Format(timeLayout))
	return err
}

func (s *Store) GetForUsers(ctx context.Context, userIDs []string) ([]Location, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	query := `SELECT user_id, latitude, longitude, accuracy_m, recorded_at, updated_at FROM locations WHERE user_id IN (` + placeholders(len(userIDs)) + `)`
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locs []Location
	for rows.Next() {
		var l Location
		var recordedAt, updatedAt string
		var accuracy sql.NullFloat64
		if err := rows.Scan(&l.UserID, &l.Latitude, &l.Longitude, &accuracy, &recordedAt, &updatedAt); err != nil {
			return nil, err
		}
		if accuracy.Valid {
			l.AccuracyM = &accuracy.Float64
		}
		l.RecordedAt, _ = time.Parse(timeLayout, recordedAt)
		l.UpdatedAt, _ = time.Parse(timeLayout, updatedAt)
		locs = append(locs, l)
	}
	return locs, rows.Err()
}

func placeholders(n int) string {
	out := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, '?')
	}
	return string(out)
}

// Settings

type Settings struct {
	UserID        string
	SharingPaused bool
}

func (s *Store) GetSettings(ctx context.Context, userID string) (*Settings, error) {
	row := s.db.QueryRowContext(ctx, `SELECT user_id, sharing_paused FROM user_settings WHERE user_id = ?`, userID)
	var st Settings
	var paused int
	if err := row.Scan(&st.UserID, &paused); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Settings{UserID: userID, SharingPaused: false}, nil
		}
		return nil, err
	}
	st.SharingPaused = paused == 1
	return &st, nil
}

func (s *Store) SetPaused(ctx context.Context, userID string, paused bool) error {
	val := 0
	if paused {
		val = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_settings (user_id, sharing_paused) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET sharing_paused = excluded.sharing_paused
	`, userID, val)
	return err
}

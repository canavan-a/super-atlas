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
	UserID      string
	DisplayName string
	AvatarID    string
	Latitude    float64
	Longitude   float64
	AccuracyM   *float64
	SpeedMps    *float64
	RecordedAt  time.Time
	UpdatedAt   time.Time
	// HereSince is when the user arrived near where they are now (zero if
	// unknown): see Upsert.
	HereSince time.Time
}

// Fix is one position reported by a user's phone (or an NPC).
type Fix struct {
	Latitude  float64
	Longitude float64
	// AccuracyM and SpeedMps are nil when the phone doesn't say.
	AccuracyM  *float64
	SpeedMps   *float64
	RecordedAt time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Upsert stores fix as userID's position unless a newer one is already
// stored, in which case ok is false. It returns when userID arrived near
// here: kept from the stored position while they stay within about 100 m of
// where they first stopped, otherwise reset to this fix.
func (s *Store) Upsert(ctx context.Context, userID string, fix Fix) (hereSince time.Time, ok bool, err error) {
	recordedAt := fix.RecordedAt.UTC().Format(timeLayout)
	var since string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO locations (user_id, latitude, longitude, accuracy_m, speed_mps, recorded_at, updated_at, here_since, here_lat, here_lng)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			accuracy_m = excluded.accuracy_m,
			speed_mps = excluded.speed_mps,
			recorded_at = excluded.recorded_at,
			updated_at = excluded.updated_at,
			here_since = CASE WHEN `+stillNear+` THEN locations.here_since ELSE excluded.here_since END,
			here_lat = CASE WHEN `+stillNear+` THEN locations.here_lat ELSE excluded.here_lat END,
			here_lng = CASE WHEN `+stillNear+` THEN locations.here_lng ELSE excluded.here_lng END
		WHERE excluded.recorded_at >= locations.recorded_at
		RETURNING here_since
	`, userID, fix.Latitude, fix.Longitude, fix.AccuracyM, fix.SpeedMps, recordedAt, time.Now().UTC().Format(timeLayout),
		recordedAt, fix.Latitude, fix.Longitude).Scan(&since)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	hereSince, _ = time.Parse(timeLayout, since)
	return hereSince, true, nil
}

// stillNear says, in an upsert, whether the new position is within about
// 100 m of the spot the stored here_since belongs to.
const stillNear = `(locations.here_since IS NOT NULL
	AND ABS(excluded.latitude - locations.here_lat) < 0.0009
	AND ABS(excluded.longitude - locations.here_lng) < 0.0012)`

const selectLocations = `SELECT l.user_id, u.display_name, COALESCE(a.id, ''), l.latitude, l.longitude, l.accuracy_m, l.speed_mps, l.recorded_at, l.updated_at, l.here_since
	FROM locations l
	JOIN users u ON u.id = l.user_id
	LEFT JOIN avatars a ON a.user_id = l.user_id
	LEFT JOIN user_settings us ON us.user_id = l.user_id
	WHERE `

func (s *Store) GetForUsers(ctx context.Context, userIDs []string) ([]Location, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}
	// Users in ghost mode are left out here, so callers never have to filter.
	return s.query(ctx, selectLocations+`COALESCE(us.sharing_paused, 0) = 0 AND l.user_id IN (`+placeholders(len(userIDs))+`)`, args...)
}

// GetOwn returns userID's own last position, ghost mode or not (it's theirs
// to see), or nil when they have none yet.
func (s *Store) GetOwn(ctx context.Context, userID string) (*Location, error) {
	locs, err := s.query(ctx, selectLocations+`l.user_id = ?`, userID)
	if err != nil || len(locs) == 0 {
		return nil, err
	}
	return &locs[0], nil
}

func (s *Store) query(ctx context.Context, query string, args ...any) ([]Location, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locs []Location
	for rows.Next() {
		var l Location
		var recordedAt, updatedAt string
		var hereSince sql.NullString
		var accuracy, speed sql.NullFloat64
		if err := rows.Scan(&l.UserID, &l.DisplayName, &l.AvatarID, &l.Latitude, &l.Longitude, &accuracy, &speed, &recordedAt, &updatedAt, &hereSince); err != nil {
			return nil, err
		}
		if accuracy.Valid {
			l.AccuracyM = &accuracy.Float64
		}
		if speed.Valid {
			l.SpeedMps = &speed.Float64
		}
		l.HereSince, _ = time.Parse(timeLayout, hereSince.String)
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

// Profile is what friends see of userID besides its position: its display
// name and profile picture id ("" for none).
func (s *Store) Profile(ctx context.Context, userID string) (name, avatarID string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT u.display_name, COALESCE(a.id, '') FROM users u LEFT JOIN avatars a ON a.user_id = u.id WHERE u.id = ?
	`, userID).Scan(&name, &avatarID)
	return name, avatarID, err
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

-- +goose Up
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE refresh_tokens (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- requester_id sent the request to addressee_id
CREATE TABLE friendships (
    id            TEXT PRIMARY KEY,
    requester_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted')),
    -- per-friend share toggle, set by each side independently once accepted
    requester_sharing INTEGER NOT NULL DEFAULT 1,
    addressee_sharing INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (requester_id <> addressee_id),
    UNIQUE (requester_id, addressee_id)
);
CREATE INDEX idx_friendships_addressee ON friendships(addressee_id);
CREATE INDEX idx_friendships_requester ON friendships(requester_id);

CREATE TABLE user_settings (
    user_id       TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- global pause ("ghost mode"): when true, no friend can see this user's location
    sharing_paused INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE locations (
    user_id     TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    latitude    REAL NOT NULL,
    longitude   REAL NOT NULL,
    accuracy_m  REAL,
    recorded_at TEXT NOT NULL,
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- +goose Down
DROP TABLE locations;
DROP TABLE user_settings;
DROP TABLE friendships;
DROP TABLE refresh_tokens;
DROP TABLE users;

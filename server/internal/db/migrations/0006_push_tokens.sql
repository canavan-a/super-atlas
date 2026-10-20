-- +goose Up
-- FCM device tokens for the Google Play build. Tied to the session that
-- registered them, so signing out (or the session expiring) drops them.
CREATE TABLE push_tokens (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_push_tokens_user ON push_tokens(user_id);

-- +goose Down
DROP TABLE push_tokens;

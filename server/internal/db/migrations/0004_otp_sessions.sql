-- +goose Up
-- Passwordless sign-in: an emailed one-time code mints a long-lived session.
DROP TABLE refresh_tokens;
DROP TABLE password_reset_codes;
ALTER TABLE users DROP COLUMN password_hash;

-- Keyed by email rather than user, since the same code also signs up a new
-- account. Only the code's hash is stored.
CREATE TABLE login_codes (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    used_at    TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_login_codes_email ON login_codes(email);

-- One row per signed-in device. expires_at slides forward as the session is
-- used, so only an idle device ever gets signed out.
CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    last_used_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at   TEXT NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE login_codes;

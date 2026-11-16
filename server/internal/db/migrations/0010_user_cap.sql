-- +goose Up
-- The allowlist gives way to a cap on accounts: anyone can sign up until
-- there are user_cap of them (superatlas-admin cap <n> raises it).
DROP TABLE allowed_emails;
CREATE TABLE app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO app_settings (key, value) VALUES ('user_cap', '150');

-- +goose Down
DROP TABLE app_settings;
CREATE TABLE allowed_emails (
    email      TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
INSERT INTO allowed_emails (email) SELECT lower(trim(email)) FROM users;

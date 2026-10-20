-- +goose Up
-- The emails allowed to sign in, managed with superatlas-admin. Nobody else
-- gets a code. Everyone who already has an account is let in.
CREATE TABLE allowed_emails (
    email      TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
INSERT INTO allowed_emails (email) SELECT lower(trim(email)) FROM users;

-- +goose Down
DROP TABLE allowed_emails;

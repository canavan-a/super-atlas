-- +goose Up
-- Profile pictures: one per user, a 256x256 JPEG re-encoded by the server.
-- id is random and changes with every new picture, so /avatars/{id} can be
-- cached forever and can't be guessed from the user's id.
CREATE TABLE avatars (
    user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    id         TEXT NOT NULL UNIQUE,
    jpeg       BLOB NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- +goose Down
DROP TABLE avatars;

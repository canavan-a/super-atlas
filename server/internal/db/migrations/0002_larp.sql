-- +goose Up
-- Debug/test feature: typing "larp" as a friend request email toggles a set
-- of simulated fake friends on/off for that user. This table tracks which
-- fake users belong to which real user so they can be cleanly torn down.
CREATE TABLE larp_fake_users (
    fake_user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    owner_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_larp_fake_users_owner ON larp_fake_users(owner_id);

-- +goose Down
DROP TABLE larp_fake_users;

-- +goose Up
-- Friends always share with each other, and ghost mode (user_settings) is the
-- only way to hide.
ALTER TABLE friendships DROP COLUMN requester_sharing;
ALTER TABLE friendships DROP COLUMN addressee_sharing;

-- Each user's personal invite link. Created on first use, replaced on reset.
ALTER TABLE users ADD COLUMN invite_code TEXT;
CREATE UNIQUE INDEX idx_users_invite_code ON users(invite_code);

-- +goose Down
DROP INDEX idx_users_invite_code;

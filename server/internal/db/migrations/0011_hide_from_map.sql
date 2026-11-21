-- +goose Up
-- Each side of a friendship can hide the other from their own map: still
-- friends (and still sharing), just not drawn. Says nothing to the other side.
ALTER TABLE friendships ADD COLUMN requester_hides INTEGER NOT NULL DEFAULT 0;
ALTER TABLE friendships ADD COLUMN addressee_hides INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE friendships DROP COLUMN addressee_hides;
ALTER TABLE friendships DROP COLUMN requester_hides;

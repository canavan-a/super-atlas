-- +goose Up
-- How fast the user was going at their last fix (when the phone knows), and
-- since when they've been near one spot: here_since is when they first came
-- within about 100 m of (here_lat, here_lng), reset whenever they leave it.
ALTER TABLE locations ADD COLUMN speed_mps REAL;
ALTER TABLE locations ADD COLUMN here_since TEXT;
ALTER TABLE locations ADD COLUMN here_lat REAL;
ALTER TABLE locations ADD COLUMN here_lng REAL;

-- +goose Down
ALTER TABLE locations DROP COLUMN here_lng;
ALTER TABLE locations DROP COLUMN here_lat;
ALTER TABLE locations DROP COLUMN here_since;
ALTER TABLE locations DROP COLUMN speed_mps;

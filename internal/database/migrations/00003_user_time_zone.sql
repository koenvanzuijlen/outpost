-- +goose Up
ALTER TABLE users
    ADD COLUMN time_zone text;

-- +goose Down
ALTER TABLE users
    DROP COLUMN time_zone;


-- +goose Up
ALTER TABLE users
    ADD COLUMN theme text NOT NULL DEFAULT 'system';

-- +goose Down
ALTER TABLE users
    DROP COLUMN theme;


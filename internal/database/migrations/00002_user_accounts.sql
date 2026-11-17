-- +goose Up
ALTER TABLE users
    ADD COLUMN role text NOT NULL DEFAULT 'user';

CREATE TABLE sessions(
    token_hash text PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at datetime NOT NULL,
    expires_at datetime NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;

ALTER TABLE users
    DROP COLUMN role;


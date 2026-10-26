-- +goose Up
CREATE TABLE users(
    id integer PRIMARY KEY,
    username text NOT NULL UNIQUE,
    password_hash text NOT NULL
);

INSERT INTO users(id, username, password_hash)
    VALUES (1, 'default', '!');

CREATE TABLE games(
    id integer PRIMARY KEY,
    igdb_id integer NOT NULL UNIQUE,
    fetched_at datetime NOT NULL,
    dlcs_fetched_at datetime,
    parent_igdb_id integer,
    title text NOT NULL,
    game_type integer NOT NULL,
    cover_url text,
    released_at datetime,
    summary text
);

CREATE INDEX idx_games_parent_igdb_id ON games(parent_igdb_id);

CREATE TABLE platforms(
    id integer PRIMARY KEY,
    igdb_id integer NOT NULL UNIQUE,
    fetched_at datetime NOT NULL,
    name text NOT NULL
);

CREATE TABLE user_games(
    id integer PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    game_id integer NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    status text,
    rating integer,
    wishlisted boolean NOT NULL DEFAULT FALSE,
    notes text,
    added_at datetime NOT NULL,
    UNIQUE (user_id, game_id),
    CHECK (NOT wishlisted OR status IS NULL),
    CHECK (rating BETWEEN 1 AND 10)
);

CREATE INDEX idx_user_games_game_id ON user_games(game_id);

CREATE TABLE owned_copies(
    id integer PRIMARY KEY,
    user_game_id integer NOT NULL REFERENCES user_games(id) ON DELETE CASCADE,
    platform_id integer NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    UNIQUE (user_game_id, platform_id)
);

CREATE TABLE owned_dlc(
    owned_copy_id integer NOT NULL REFERENCES owned_copies(id) ON DELETE CASCADE,
    dlc_game_id integer NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    PRIMARY KEY (owned_copy_id, dlc_game_id)
);

CREATE INDEX idx_owned_dlc_dlc_game_id ON owned_dlc(dlc_game_id);

CREATE TABLE playthroughs(
    id integer PRIMARY KEY,
    user_game_id integer NOT NULL REFERENCES user_games(id) ON DELETE CASCADE,
    platform_id integer REFERENCES platforms(id) ON DELETE RESTRICT,
    name text,
    outcome text NOT NULL,
    started_at datetime,
    finished_at datetime
);

CREATE INDEX idx_playthroughs_user_game_id ON playthroughs(user_game_id);

-- +goose Down
DROP TABLE playthroughs;

DROP TABLE owned_dlc;

DROP TABLE owned_copies;

DROP TABLE user_games;

DROP TABLE platforms;

DROP TABLE games;

DROP TABLE users;


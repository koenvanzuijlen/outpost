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
    parent_igdb_id integer,
    title text NOT NULL,
    game_type integer NOT NULL,
    cover_url text,
    release_year integer,
    summary text
);

CREATE INDEX idx_games_parent_igdb_id ON games(parent_igdb_id);

CREATE TABLE platforms(
    id integer PRIMARY KEY,
    igdb_id integer NOT NULL UNIQUE,
    fetched_at datetime NOT NULL,
    name text NOT NULL
);

CREATE TABLE collection_items(
    id integer PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    game_id integer NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    platform_id integer NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    status text NOT NULL,
    rating integer,
    started_at datetime,
    finished_at datetime,
    notes text,
    UNIQUE (user_id, game_id, platform_id)
);

CREATE INDEX idx_collection_items_game_id ON collection_items(game_id);

CREATE TABLE dlc_ownership(
    id integer PRIMARY KEY,
    collection_item_id integer NOT NULL REFERENCES collection_items(id) ON DELETE CASCADE,
    dlc_game_id integer NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    owned integer NOT NULL DEFAULT 0,
    tracked_collection_item_id integer REFERENCES collection_items(id) ON DELETE SET NULL,
    UNIQUE (collection_item_id, dlc_game_id)
);

CREATE INDEX idx_dlc_ownership_tracked ON dlc_ownership(tracked_collection_item_id);

CREATE INDEX idx_dlc_ownership_dlc_game_id ON dlc_ownership(dlc_game_id);

-- +goose Down
DROP TABLE dlc_ownership;

DROP TABLE collection_items;

DROP TABLE platforms;

DROP TABLE games;

DROP TABLE users;


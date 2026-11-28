-- +goose Up
CREATE TABLE game_lists(
    id integer PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL COLLATE NOCASE,
    description text,
    UNIQUE (user_id, name),
    CHECK (name <> ''),
    CHECK (description <> '')
);

CREATE TABLE game_list_items(
    game_list_id integer NOT NULL REFERENCES game_lists(id) ON DELETE CASCADE,
    game_id integer NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    added_at datetime NOT NULL,
    PRIMARY KEY (game_list_id, game_id)
);

CREATE INDEX idx_game_list_items_game_id ON game_list_items(game_id);

-- +goose Down
DROP TABLE game_list_items;

DROP TABLE game_lists;


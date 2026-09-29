-- name: GetGameByIGDBID :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = ?
LIMIT 1;

-- name: CreateGame :one
INSERT INTO games(igdb_id, fetched_at, parent_igdb_id, title, game_type, cover_url, release_year, summary)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING
    *;


-- name: GetGameByIGDBID :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = ?
LIMIT 1;

-- name: ListDLCGames :many
SELECT
    *
FROM
    games
WHERE
    parent_igdb_id = ?
    AND game_type IN (sqlc.slice(game_types))
ORDER BY
    released_at ASC NULLS LAST,
    title;

-- name: UpsertGame :one
INSERT INTO games(igdb_id, fetched_at, parent_igdb_id, title, game_type, cover_url, released_at, summary)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (igdb_id)
    DO UPDATE SET
        fetched_at = excluded.fetched_at,
        parent_igdb_id = excluded.parent_igdb_id,
        title = excluded.title,
        game_type = excluded.game_type,
        cover_url = excluded.cover_url,
        released_at = excluded.released_at,
        summary = excluded.summary
    RETURNING
        *;

-- name: MarkDLCsFetched :execrows
UPDATE
    games
SET
    dlcs_fetched_at = ?
WHERE
    igdb_id = ?;


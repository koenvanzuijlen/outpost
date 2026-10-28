-- name: GetGameByIGDBID :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = ?
LIMIT 1;

-- name: GetDLCGame :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = sqlc.arg(dlc_igdb_id)
    AND parent_igdb_id = sqlc.arg(parent_igdb_id)
    AND game_type IN (sqlc.slice(game_types));

-- name: ListDLCGames :many
SELECT
    sqlc.embed(g),
    ug.id IS NOT NULL AS in_library,
    ug.status
FROM
    games g
    LEFT JOIN user_games ug ON g.id = ug.game_id
        AND ug.user_id = sqlc.arg(user_id)
WHERE
    g.parent_igdb_id = sqlc.arg(parent_igdb_id)
    AND g.game_type IN (sqlc.slice(game_types))
ORDER BY
    g.released_at ASC NULLS LAST,
    g.title;

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


-- name: GetGameByIGDBID :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = ?
LIMIT 1;

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


-- name: GetGameByIGDBID :one
SELECT
    *
FROM
    games
WHERE
    igdb_id = ?
LIMIT 1;

-- name: UpsertGame :one
INSERT INTO games(igdb_id, fetched_at, parent_igdb_id, title, game_type, cover_url, release_year)
    VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (igdb_id)
    DO UPDATE SET
        fetched_at = EXCLUDED.fetched_at,
        parent_igdb_id = EXCLUDED.parent_igdb_id,
        title = EXCLUDED.title,
        game_type = EXCLUDED.game_type,
        cover_url = EXCLUDED.cover_url,
        release_year = EXCLUDED.release_year
    RETURNING
        *;


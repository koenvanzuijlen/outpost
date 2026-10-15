-- name: ListPlatformsForUserGameID :many
SELECT
    p.*
FROM
    platforms p
    JOIN owned_copies oc ON oc.platform_id = p.id
WHERE
    oc.user_game_id = ?
ORDER BY
    p.name;

-- name: GetPlatformByIGDBID :one
SELECT
    *
FROM
    platforms
WHERE
    igdb_id = ?;

-- name: UpsertPlatform :one
INSERT INTO platforms(igdb_id, fetched_at, name)
    VALUES (?, ?, ?)
ON CONFLICT (igdb_id)
    DO UPDATE SET
        fetched_at = excluded.fetched_at,
        name = excluded.name
    RETURNING
        *;


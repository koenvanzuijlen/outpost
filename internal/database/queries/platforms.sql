-- name: ListPlatformsForGame :many
SELECT
    p.*
FROM
    platforms p
    JOIN game_platforms gp ON gp.platform_id = p.id
WHERE
    gp.game_id = ?
ORDER BY
    p.name;

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

-- name: ListPlatformsForOwnedDLC :many
SELECT
    p.*
FROM
    owned_dlc od
    JOIN owned_copies oc ON oc.id = od.owned_copy_id
    JOIN user_games ug ON ug.id = oc.user_game_id
    JOIN platforms p ON p.id = oc.platform_id
WHERE
    od.dlc_game_id = ?
    AND ug.user_id = ?
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

-- name: CreateGamePlatform :exec
INSERT INTO game_platforms(game_id, platform_id)
    VALUES (?, ?)
ON CONFLICT (game_id, platform_id)
    DO NOTHING;

-- name: DeleteGamePlatforms :exec
DELETE FROM game_platforms
WHERE game_id = ?;


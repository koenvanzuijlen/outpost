-- name: ListOwnedCopyDLCs :many
SELECT
    p.igdb_id AS platform_igdb_id,
    sqlc.embed(g),
    od.dlc_game_id IS NOT NULL AS owned
FROM
    owned_copies oc
    JOIN platforms p ON p.id = oc.platform_id
    JOIN game_platforms gp ON gp.platform_id = oc.platform_id
    JOIN games g ON g.id = gp.game_id
    LEFT JOIN owned_dlc od ON od.owned_copy_id = oc.id
        AND od.dlc_game_id = g.id
WHERE
    oc.user_game_id = ?
    AND g.parent_igdb_id = ?
    AND g.game_type IN (sqlc.slice(game_types))
ORDER BY
    g.released_at ASC NULLS LAST,
    g.title;

-- name: GetCopyDLC :one
SELECT
    oc.id AS owned_copy_id,
    g.id AS dlc_game_id
FROM
    owned_copies oc
    JOIN platforms p ON p.id = oc.platform_id
    JOIN game_platforms gp ON gp.platform_id = oc.platform_id
    JOIN games g ON g.id = gp.game_id
WHERE
    oc.user_game_id = sqlc.arg(user_game_id)
    AND p.igdb_id = sqlc.arg(platform_igdb_id)
    AND g.igdb_id = sqlc.arg(dlc_igdb_id)
    AND g.parent_igdb_id = sqlc.arg(parent_igdb_id)
    AND g.game_type IN (sqlc.slice(game_types));

-- name: CreateOwnedDLC :exec
INSERT INTO owned_dlc(owned_copy_id, dlc_game_id)
    VALUES (?, ?)
ON CONFLICT
    DO NOTHING;

-- name: DeleteOwnedDLC :exec
DELETE FROM owned_dlc
WHERE owned_copy_id = ?
    AND dlc_game_id = ?;


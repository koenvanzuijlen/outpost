-- name: ListOwnedCopiesForUserID :many
SELECT
    oc.id,
    oc.user_game_id,
    sqlc.embed(p)
FROM
    owned_copies oc
    JOIN user_games ug ON ug.id = oc.user_game_id
    JOIN platforms p ON p.id = oc.platform_id
WHERE
    ug.user_id = ?
ORDER BY
    p.name,
    p.igdb_id;

-- name: CreateOwnedCopy :one
INSERT INTO owned_copies(user_game_id, platform_id)
    VALUES (?, ?)
RETURNING
    *;

-- name: DeleteOwnedCopy :execrows
DELETE FROM owned_copies
WHERE user_game_id = ?
    AND platform_id =(
        SELECT
            id
        FROM
            platforms
        WHERE
            igdb_id = ?);


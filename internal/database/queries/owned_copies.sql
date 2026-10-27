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


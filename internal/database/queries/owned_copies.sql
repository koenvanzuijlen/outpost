-- name: ListOwnedCopies :many
SELECT
    sqlc.embed(oc),
    sqlc.embed(ug),
    sqlc.embed(g),
    sqlc.embed(p)
FROM
    owned_copies oc
    JOIN user_games ug ON oc.user_game_id = ug.id
    JOIN games g ON ug.game_id = g.id
    JOIN platforms p ON oc.platform_id = p.id
WHERE
    ug.user_id = ?
ORDER BY
    g.title;

-- name: CreateOwnedCopy :one
INSERT INTO owned_copies(user_game_id, platform_id)
    VALUES (?, ?)
RETURNING
    *;


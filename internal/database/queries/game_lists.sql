-- name: GetGameList :one
SELECT
    *
FROM
    game_lists
WHERE
    id = ?
    AND user_id = ?;

-- name: ListGameListsForUser :many
SELECT
    *
FROM
    game_lists
WHERE
    user_id = ?
ORDER BY
    name,
    id;

-- name: CreateGameList :one
INSERT INTO game_lists(user_id, name, description)
    VALUES (?, ?, ?)
RETURNING
    *;

-- name: UpdateGameList :execrows
UPDATE
    game_lists
SET
    name = ?,
    description = ?
WHERE
    id = ?
    AND user_id = ?;

-- name: DeleteGameList :exec
DELETE FROM game_lists
WHERE id = ?
    AND user_id = ?;


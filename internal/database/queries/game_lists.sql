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

-- name: ListGameListsForGame :many
SELECT
    sqlc.embed(gl),
    gli.game_id IS NOT NULL AS has_game
FROM
    game_lists gl
    LEFT JOIN game_list_items gli ON gl.id = gli.game_list_id
        AND gli.game_id = sqlc.arg(game_id)
WHERE
    gl.user_id = sqlc.arg(user_id)
ORDER BY
    gl.name,
    gl.id;

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


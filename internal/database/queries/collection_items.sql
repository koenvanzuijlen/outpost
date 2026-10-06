-- name: ListCollectionItemsWithGames :many
SELECT
    sqlc.embed(ci),
    sqlc.embed(g)
FROM
    collection_items ci
    JOIN games g ON ci.game_id = g.id
WHERE
    ci.user_id = ?
ORDER BY
    g.title;

-- name: CreateCollectionItem :one
INSERT INTO collection_items(user_id, game_id, platform, status)
    VALUES (?, ?, ?, ?)
RETURNING
    *;


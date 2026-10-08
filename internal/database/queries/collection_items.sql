-- name: ListCollectionItemsWithGames :many
SELECT
    sqlc.embed(ci),
    sqlc.embed(g),
    sqlc.embed(p)
FROM
    collection_items ci
    JOIN games g ON ci.game_id = g.id
    JOIN platforms p ON ci.platform_id = p.id
WHERE
    ci.user_id = ?
ORDER BY
    g.title;

-- name: CreateCollectionItem :one
INSERT INTO collection_items(user_id, game_id, platform_id, status)
    VALUES (?, ?, ?, ?)
RETURNING
    *;


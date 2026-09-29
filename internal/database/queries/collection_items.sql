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


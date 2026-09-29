-- name: ListCollectionItems :many
SELECT
    *
FROM
    collection_items
WHERE
    user_id = ?;


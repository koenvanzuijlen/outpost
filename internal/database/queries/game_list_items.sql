-- name: ListGameListEntries :many
SELECT
    sqlc.embed(g),
    gli.added_at,
    ug.status,
    ug.rating
FROM
    game_list_items gli
    JOIN game_lists gl ON gl.id = gli.game_list_id
    JOIN games g ON g.id = gli.game_id
    LEFT JOIN user_games ug ON ug.game_id = gli.game_id
        AND ug.user_id = gl.user_id
        AND NOT ug.wishlisted
WHERE
    gli.game_list_id = sqlc.arg(game_list_id)
    AND gl.user_id = sqlc.arg(user_id)
ORDER BY
    g.title,
    g.igdb_id;

-- name: ListGameListItemCoversForUser :many
SELECT
    gli.game_list_id,
    g.cover_url
FROM
    game_list_items gli
    JOIN game_lists gl ON gl.id = gli.game_list_id
    JOIN games g ON g.id = gli.game_id
WHERE
    gl.user_id = ?
ORDER BY
    gli.added_at DESC,
    g.title,
    g.igdb_id;

-- name: CreateGameListItem :exec
INSERT INTO game_list_items(game_list_id, game_id, added_at)
    VALUES (?, ?, ?)
ON CONFLICT (game_list_id, game_id)
    DO NOTHING;

-- name: DeleteGameListItem :exec
DELETE FROM game_list_items
WHERE game_list_id IN (
        SELECT
            gl.id
        FROM
            game_lists gl
        WHERE
            gl.id = sqlc.arg(game_list_id)
            AND gl.user_id = sqlc.arg(user_id))
    AND game_id IN (
        SELECT
            g.id
        FROM
            games g
        WHERE
            g.igdb_id = sqlc.arg(igdb_id));


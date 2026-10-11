-- name: ListWishlistedUserGames :many
SELECT
    sqlc.embed(ug),
    sqlc.embed(g)
FROM
    user_games ug
    JOIN games g ON ug.game_id = g.id
WHERE
    ug.user_id = ?
    AND ug.wishlisted = TRUE
ORDER BY
    g.released_at ASC NULLS LAST;

-- name: CreateWishlistedUserGame :execrows
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, NULL, TRUE, ?)
ON CONFLICT (user_id, game_id)
    DO NOTHING;

-- name: UpsertOwnedUserGame :one
INSERT INTO user_games(user_id, game_id, status, wishlisted, added_at)
    VALUES (?, ?, ?, FALSE, ?)
ON CONFLICT (user_id, game_id)
    DO UPDATE SET
        status = COALESCE(user_games.status, excluded.status),
        wishlisted = FALSE
    RETURNING
        *;

